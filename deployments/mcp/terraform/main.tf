# Copyright 2026 Retail Cortex
#
# Terraform Infrastructure Provisioning for AI Daily Brief MCP Server Infrastructure
# Supports Multi-Environment deployments (dev, prod) on Cloud Run & AlloyDB

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }
}

provider "google" {
  project = var.project_id != "" ? var.project_id : null
  region  = var.region
}

data "google_client_config" "default" {}

locals {
  project_id        = var.project_id != "" ? var.project_id : data.google_client_config.default.project
  service_full_name = "${var.service_name}-${var.environment}"
  alloydb_secret_id = "ai-daily-brief-db-url-${var.environment}"
  enc_key_secret_id = "ai-daily-brief-encryption-key-${var.environment}"
}

# -----------------------------------------------------------------------------
# 1. Enable Required GCP APIs
# -----------------------------------------------------------------------------
locals {
  services = [
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "aiplatform.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "iam.googleapis.com",
    "alloydb.googleapis.com",
    "compute.googleapis.com",
    "servicenetworking.googleapis.com",
  ]
}

resource "google_project_service" "apis" {
  for_each           = toset(local.services)
  project            = local.project_id
  service            = each.value
  disable_on_destroy = false
}

# -----------------------------------------------------------------------------
# 2. Artifact Registry for Container Images
# -----------------------------------------------------------------------------
resource "google_artifact_registry_repository" "repo" {
  depends_on    = [google_project_service.apis]
  project       = local.project_id
  location      = var.region
  repository_id = "ai-daily-brief"
  description   = "Docker repository for AI Daily Brief microservices"
  format        = "DOCKER"
}

# -----------------------------------------------------------------------------
# 3. Build & Push Container Image
# -----------------------------------------------------------------------------
locals {
  image_url = "${var.region}-docker.pkg.dev/${local.project_id}/${google_artifact_registry_repository.repo.name}/${var.service_name}:${var.image_tag}"
}

resource "null_resource" "build_image" {
  count = var.build_image_on_apply ? 1 : 0

  depends_on = [
    google_artifact_registry_repository.repo,
    google_project_service.apis,
  ]

  triggers = {
    image_url = local.image_url
  }

  provisioner "local-exec" {
    working_dir = "${path.module}/../../.."
    command     = "gcloud builds submit --project=${local.project_id} --config=deployments/mcp/cloudbuild.yaml --substitutions=_IMAGE_URL=${local.image_url} ."
  }
}

# -----------------------------------------------------------------------------
# 4. VPC Network & Private Service Networking (for AlloyDB)
# -----------------------------------------------------------------------------
resource "google_compute_network" "vpc" {
  count                   = var.enable_alloydb ? 1 : 0
  depends_on              = [google_project_service.apis]
  project                 = local.project_id
  name                    = "${var.vpc_network_name}-${var.environment}"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  count                    = var.enable_alloydb ? 1 : 0
  project                  = local.project_id
  name                     = "${var.vpc_subnet_name}-${var.environment}"
  region                   = var.region
  network                  = google_compute_network.vpc[0].id
  ip_cidr_range            = var.vpc_subnet_cidr
  private_ip_google_access = true
}

resource "google_compute_global_address" "private_ip_alloc" {
  count         = var.enable_alloydb ? 1 : 0
  project       = local.project_id
  name          = "alloydb-private-ip-${var.environment}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc[0].id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  count                   = var.enable_alloydb ? 1 : 0
  network                 = google_compute_network.vpc[0].id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_alloc[0].name]
}

# -----------------------------------------------------------------------------
# 5. AlloyDB Cluster & Primary Instance
# -----------------------------------------------------------------------------
resource "random_password" "alloydb_password" {
  count   = var.enable_alloydb ? 1 : 0
  length  = 24
  special = false
}

resource "google_alloydb_cluster" "cluster" {
  count      = var.enable_alloydb ? 1 : 0
  project    = local.project_id
  cluster_id = "${var.alloydb_cluster_id}-${var.environment}"
  location   = var.region
  network_config {
    network = google_compute_network.vpc[0].id
  }

  initial_user {
    user     = var.alloydb_user_name
    password = random_password.alloydb_password[0].result
  }

  depends_on = [google_service_networking_connection.private_vpc_connection]
}

resource "google_alloydb_instance" "primary" {
  count         = var.enable_alloydb ? 1 : 0
  cluster       = google_alloydb_cluster.cluster[0].name
  instance_id   = "${var.alloydb_instance_id}-${var.environment}"
  instance_type = "PRIMARY"

  machine_config {
    cpu_count = var.alloydb_cpu_count
  }
}

locals {
  alloydb_generated_dsn = var.enable_alloydb ? "postgres://${var.alloydb_user_name}:${random_password.alloydb_password[0].result}@${google_alloydb_instance.primary[0].ip_address}:5432/${var.alloydb_database_name}?sslmode=require" : ""
  effective_db_url      = var.alloydb_database_url != "" ? var.alloydb_database_url : local.alloydb_generated_dsn
}

# -----------------------------------------------------------------------------
# 6. Dedicated Service Account & IAM Permissions
# -----------------------------------------------------------------------------
resource "google_service_account" "sa" {
  project      = local.project_id
  account_id   = "ai-daily-brief-sa-${var.environment}"
  display_name = "AI Daily Brief Cloud Run Service Account (${var.environment})"
}

resource "google_project_iam_member" "secret_accessor" {
  project = local.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.sa.email}"
}

resource "google_project_iam_member" "vertex_user" {
  project = local.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.sa.email}"
}

# -----------------------------------------------------------------------------
# 7. Secret Manager: AES Key & Database DSN
# -----------------------------------------------------------------------------
resource "random_password" "encryption_key" {
  length  = 32
  special = false
}

resource "google_secret_manager_secret" "encryption_key" {
  depends_on          = [google_project_service.apis]
  project             = local.project_id
  secret_id           = local.enc_key_secret_id
  deletion_protection = false

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "encryption_key_val" {
  secret      = google_secret_manager_secret.encryption_key.id
  secret_data = random_password.encryption_key.result
}

resource "google_secret_manager_secret" "db_url" {
  depends_on          = [google_project_service.apis]
  project             = local.project_id
  secret_id           = local.alloydb_secret_id
  deletion_protection = false

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "db_url_val" {
  count       = local.effective_db_url != "" ? 1 : 0
  secret      = google_secret_manager_secret.db_url.id
  secret_data = local.effective_db_url
}

# -----------------------------------------------------------------------------
# 8. Cloud Run v2 Service (MCP Server Control Plane)
# -----------------------------------------------------------------------------
resource "google_cloud_run_v2_service" "mcp_server" {
  depends_on = [
    google_project_service.apis,
    google_project_iam_member.secret_accessor,
    google_project_iam_member.vertex_user,
    null_resource.build_image,
  ]

  name                = local.service_full_name
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false

  template {
    service_account = google_service_account.sa.email

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    dynamic "vpc_access" {
      for_each = var.enable_alloydb ? [1] : (var.vpc_network != "" && var.vpc_subnetwork != "" ? [1] : [])
      content {
        network_interfaces {
          network    = var.enable_alloydb ? google_compute_network.vpc[0].id : var.vpc_network
          subnetwork = var.enable_alloydb ? google_compute_subnetwork.subnet[0].id : var.vpc_subnetwork
        }
        egress = "PRIVATE_RANGES_ONLY"
      }
    }

    containers {
      image = local.image_url

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "1Gi"
        }
        cpu_idle = true
      }

      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = local.project_id
      }

      env {
        name  = "MODENV_PROFILE"
        value = var.environment
      }

      dynamic "env" {
        for_each = local.effective_db_url != "" ? [1] : []
        content {
          name = "ALLOYDB_DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.db_url.secret_id
              version = "latest"
            }
          }
        }
      }

      startup_probe {
        initial_delay_seconds = 2
        period_seconds        = 5
        failure_threshold     = 5
        tcp_socket {
          port = 8080
        }
      }

      liveness_probe {
        period_seconds    = 15
        failure_threshold = 3
        http_get {
          path = "/healthz"
          port = 8080
        }
      }
    }
  }
}

# -----------------------------------------------------------------------------
# 9. IAM Policy for Cloud Run Ingress
# -----------------------------------------------------------------------------
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  count    = var.allow_unauthenticated ? 1 : 0
  project  = google_cloud_run_v2_service.mcp_server.project
  location = google_cloud_run_v2_service.mcp_server.location
  name     = google_cloud_run_v2_service.mcp_server.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
