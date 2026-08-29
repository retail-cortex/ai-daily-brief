# Copyright 2026 Retail Cortex
#
# Terraform Variables for AI Daily Brief Cloud Infrastructure

variable "environment" {
  type        = string
  description = "Target deployment environment profile (e.g. dev, prod, integration)"
  default     = "dev"
}

variable "project_id" {
  type        = string
  description = "Google Cloud Project ID (optional: auto-detected from active gcloud config if omitted)"
  default     = ""
}

variable "region" {
  type        = string
  description = "Google Cloud primary region for Cloud Run, AlloyDB, and resources"
  default     = "us-central1"
}

variable "service_name" {
  type        = string
  description = "Base Cloud Run service name for the MCP server"
  default     = "ai-daily-brief-mcp"
}

variable "image_tag" {
  type        = string
  description = "Container image tag for deployment"
  default     = "latest"
}

variable "build_image_on_apply" {
  type        = bool
  description = "Automatically build and push container image via Cloud Build before deploying Cloud Run"
  default     = true
}

variable "min_instances" {
  type        = number
  description = "Minimum instance count for Cloud Run (0 for scale-to-zero in dev)"
  default     = 0
}

variable "max_instances" {
  type        = number
  description = "Maximum instance count for Cloud Run"
  default     = 10
}

variable "allow_unauthenticated" {
  type        = bool
  description = "Allow public unauthenticated access to the Cloud Run service"
  default     = true
}

# -----------------------------------------------------------------------------
# AlloyDB & VPC Configuration
# -----------------------------------------------------------------------------
variable "enable_alloydb" {
  type        = bool
  description = "Whether to provision managed Google Cloud AlloyDB cluster and instance"
  default     = false
}

variable "alloydb_cluster_id" {
  type        = string
  description = "Cluster identifier for AlloyDB"
  default     = "ai-daily-brief-cluster"
}

variable "alloydb_instance_id" {
  type        = string
  description = "Primary instance identifier for AlloyDB"
  default     = "ai-daily-brief-primary"
}

variable "alloydb_database_name" {
  type        = string
  description = "Database name within AlloyDB"
  default     = "aibrief"
}

variable "alloydb_user_name" {
  type        = string
  description = "Database master username for AlloyDB"
  default     = "aibrief"
}

variable "alloydb_cpu_count" {
  type        = number
  description = "CPU core count for AlloyDB primary instance (2, 4, 8, etc.)"
  default     = 2
}

variable "alloydb_database_url" {
  type        = string
  description = "Custom AlloyDB / PostgreSQL connection string override (if not provisioning new AlloyDB)"
  sensitive   = true
  default     = ""
}

variable "vpc_network_name" {
  type        = string
  description = "VPC network name for AlloyDB private connectivity and Cloud Run Direct VPC Egress"
  default     = "ai-daily-brief-vpc"
}

variable "vpc_subnet_name" {
  type        = string
  description = "VPC subnetwork name for Cloud Run Direct VPC Egress"
  default     = "ai-daily-brief-subnet"
}

variable "vpc_subnet_cidr" {
  type        = string
  description = "CIDR range for the VPC subnetwork"
  default     = "10.0.0.0/24"
}

variable "vpc_network" {
  type        = string
  description = "Existing VPC network resource URI (optional override when enable_alloydb is false)"
  default     = ""
}

variable "vpc_subnetwork" {
  type        = string
  description = "Existing VPC subnetwork resource URI (optional override when enable_alloydb is false)"
  default     = ""
}
