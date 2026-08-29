# Copyright 2026 Retail Cortex
#
# Terraform Variables for AI Daily Brief A2A Agent Infrastructure

variable "project_id" {
  description = "Google Cloud Project ID where resources will be provisioned"
  type        = string
  default     = ""
}

variable "region" {
  description = "Google Cloud region for deployment"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Deployment environment (e.g. 'dev', 'prod')"
  type        = string
  default     = "dev"
}

variable "service_name" {
  description = "Base name for the A2A Agent Cloud Run service"
  type        = string
  default     = "ai-daily-brief-agent"
}

variable "mcp_service_name" {
  description = "Name of the target MCP Server Cloud Run service"
  type        = string
  default     = "ai-daily-brief-mcp"
}

variable "mcp_service_url" {
  description = "Explicit URL of the MCP Server. If empty, discovers from Cloud Run service name."
  type        = string
  default     = ""
}

variable "image_tag" {
  description = "Container image tag to deploy"
  type        = string
  default     = "latest"
}

variable "build_image_on_apply" {
  description = "Whether to trigger Google Cloud Build during terraform apply"
  type        = bool
  default     = false
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances (0 for scale-to-zero)"
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances"
  type        = number
  default     = 5
}

variable "allow_unauthenticated" {
  description = "Whether to allow unauthenticated public access to the A2A Agent endpoint"
  type        = bool
  default     = false
}
