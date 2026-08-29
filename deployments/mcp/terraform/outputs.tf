# Copyright 2026 Retail Cortex
#
# Terraform Outputs for AI Daily Brief Infrastructure

output "environment" {
  description = "Target deployment environment"
  value       = var.environment
}

output "project_id" {
  description = "Google Cloud Project ID"
  value       = local.project_id
}

output "cloud_run_url" {
  description = "Public HTTPS URL for the deployed Cloud Run MCP server"
  value       = google_cloud_run_v2_service.mcp_server.uri != "" ? google_cloud_run_v2_service.mcp_server.uri : (length(google_cloud_run_v2_service.mcp_server.urls) > 0 ? google_cloud_run_v2_service.mcp_server.urls[0] : "")
}

output "cloud_run_service_name" {
  description = "Full service name of the deployed Cloud Run service"
  value       = google_cloud_run_v2_service.mcp_server.name
}

output "artifact_registry_repo" {
  description = "Artifact Registry repository URI for container images"
  value       = "${var.region}-docker.pkg.dev/${local.project_id}/${google_artifact_registry_repository.repo.name}"
}

output "service_account_email" {
  description = "Dedicated runtime service account email"
  value       = google_service_account.sa.email
}

output "encryption_key_secret_id" {
  description = "Secret Manager secret ID for AES encryption key"
  value       = google_secret_manager_secret.encryption_key.secret_id
}

output "alloydb_cluster_id" {
  description = "AlloyDB cluster ID (if provisioned)"
  value       = var.enable_alloydb ? google_alloydb_cluster.cluster[0].cluster_id : null
}

output "alloydb_primary_ip" {
  description = "AlloyDB primary instance private IP address (if provisioned)"
  value       = var.enable_alloydb ? google_alloydb_instance.primary[0].ip_address : null
}

output "alloydb_secret_id" {
  description = "Secret Manager secret ID for the database DSN"
  value       = google_secret_manager_secret.db_url.secret_id
}
