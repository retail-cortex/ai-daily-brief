# Copyright 2026 Retail Cortex
#
# Terraform Outputs for AI Daily Brief A2A Agent Infrastructure

output "service_name" {
  description = "The name of the deployed A2A Agent Cloud Run service"
  value       = google_cloud_run_v2_service.agent_service.name
}

output "service_url" {
  description = "The HTTPS URL of the deployed A2A Agent Cloud Run service"
  value       = google_cloud_run_v2_service.agent_service.uri
}

output "service_account_email" {
  description = "Dedicated service account email used by the A2A Agent"
  value       = google_service_account.sa.email
}

output "mcp_target_url" {
  description = "The target MCP Server URL connected to this agent"
  value       = local.effective_mcp_url
}
