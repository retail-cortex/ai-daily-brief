# Copyright 2026 Retail Cortex
#
# Terraform Infrastructure Provisioning for AI Daily Brief A2A Agent Infrastructure
# Supports Multi-Environment deployments (dev, prod) on Cloud Run with Vertex AI

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
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
  mcp_full_name     = "${var.mcp_service_name}-${var.environment}"
  image_url         = "${var.region}-docker.pkg.dev/${local.project_id}/ai-daily-brief/${var.service_name}:${var.image_tag}"
}

# -----------------------------------------------------------------------------
# 1. Discover Target MCP Server URL (if not explicitly overridden)
# -----------------------------------------------------------------------------
data "google_cloud_run_v2_service" "mcp_server" {
  count    = var.mcp_service_url == "" ? 1 : 0
  project  = local.project_id
  location = var.region
  name     = local.mcp_full_name
}

locals {
  effective_mcp_url = var.mcp_service_url != "" ? var.mcp_service_url : (
    length(data.google_cloud_run_v2_service.mcp_server) > 0 ? data.google_cloud_run_v2_service.mcp_server[0].uri : ""
  )
}

# -----------------------------------------------------------------------------
# 2. Build & Push Container Image (Optional on apply)
# -----------------------------------------------------------------------------
resource "null_resource" "build_image" {
  count = var.build_image_on_apply ? 1 : 0

  triggers = {
    image_url = local.image_url
  }

  provisioner "local-exec" {
    command = "gcloud builds submit --project=${local.project_id} --config=${path.root}/../cloudbuild.yaml --substitutions=_IMAGE_URL=${local.image_url} ${path.root}/../../.."
  }
}

# -----------------------------------------------------------------------------
# 3. IAM Service Account for A2A Agent
# -----------------------------------------------------------------------------
resource "google_service_account" "agent_sa" {
  project      = local.project_id
  account_id   = "${var.service_name}-sa-${var.environment}"
  display_name = "AI Daily Brief A2A Agent Service Account (${var.environment})"
}

# Grant Vertex AI user role for reasoning / LLM generation
resource "google_project_iam_member" "vertex_ai_user" {
  project = local.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.agent_sa.email}"
}

# Grant Cloud Run Invoker on target MCP Server service
resource "google_cloud_run_v2_service_iam_member" "mcp_invoker" {
  project  = local.project_id
  location = var.region
  name     = local.mcp_full_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.agent_sa.email}"
}

# -----------------------------------------------------------------------------
# 4. Deploy A2A Agent Service on Cloud Run
# -----------------------------------------------------------------------------
resource "google_cloud_run_v2_service" "agent_service" {
  name     = local.service_full_name
  location = var.region
  project  = local.project_id

  depends_on = [
    null_resource.build_image,
    google_service_account.agent_sa,
  ]

  template {
    service_account = google_service_account.agent_sa.email

    scaling {
      min_instance_count = var.min_instance_count
      max_instance_count = var.max_instance_count
    }

    containers {
      image = local.image_url

      resources {
        limits = {
          cpu    = var.cpu_limit
          memory = var.memory_limit
        }
      }

      ports {
        container_port = 8080
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
        for_each = local.effective_mcp_url != "" ? [1] : []
        content {
          name  = "MCP_SERVER_URL"
          value = local.effective_mcp_url
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
# 5. Optional IAM Policy for Public / Ingress Access & Discovery Engine Invoker
# -----------------------------------------------------------------------------
resource "google_cloud_run_v2_service_iam_member" "public_access" {
  count    = var.allow_unauthenticated ? 1 : 0
  project  = google_cloud_run_v2_service.agent_service.project
  location = google_cloud_run_v2_service.agent_service.location
  name     = google_cloud_run_v2_service.agent_service.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

data "google_project" "current" {
  project_id = local.project_id
}

resource "google_cloud_run_v2_service_iam_member" "discovery_engine_invoker" {
  project  = google_cloud_run_v2_service.agent_service.project
  location = google_cloud_run_v2_service.agent_service.location
  name     = google_cloud_run_v2_service.agent_service.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-discoveryengine.iam.gserviceaccount.com"
}
