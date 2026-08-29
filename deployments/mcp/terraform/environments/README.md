# Multi-Environment Terraform Deployment Guide (MCP Server)

This directory provides environment-specific variables for provisioning the AI Daily Brief MCP Server platform across environments (`dev`, `prod`).

## Architecture & Configuration Mapping

| Component | Dev (`dev.tfvars`) | Prod (`prod.tfvars`) |
| :--- | :--- | :--- |
| **Config Profile (`MODENV_PROFILE`)** | `dev` (`configs/mcp/.env.dev.toml`) | `prod` (`configs/mcp/.env.prod.toml`) |
| **Cloud Run Service** | `ai-daily-brief-mcp-dev` | `ai-daily-brief-mcp-prod` |
| **Min Instances** | `0` (scale-to-zero) | `1` (warm instance) |
| **Max Instances** | `5` | `20` |
| **Auth Mode** | Private IAM (`roles/run.invoker`) | Private IAM (`roles/run.invoker`) |
| **AlloyDB Provisioning** | Optional / Standalone fallback | Managed AlloyDB Cluster + Instance |
| **AlloyDB CPU Cores** | `2` vCPUs | `4` vCPUs |
| **VPC CIDR Range** | `10.10.0.0/24` | `10.20.0.0/24` |
| **Secret Manager Secrets** | `*-dev` | `*-prod` |

---

## Deployment Commands

### 1. Development Environment
```bash
cd deployments/mcp/terraform

# Initialize Terraform (first time)
terraform init

# Plan Dev Deployment
terraform plan -var-file=environments/dev.tfvars

# Apply Dev Deployment
terraform apply -var-file=environments/dev.tfvars
```

### 2. Production Environment
```bash
cd deployments/mcp/terraform

# Plan Prod Deployment
terraform plan -var-file=environments/prod.tfvars

# Apply Prod Deployment
terraform apply -var-file=environments/prod.tfvars
```
