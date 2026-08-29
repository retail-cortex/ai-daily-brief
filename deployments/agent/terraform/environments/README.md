# Multi-Environment Terraform Deployment Guide (A2A Agent)

This directory provides environment-specific variables for provisioning the AI Daily Brief A2A Agent on Google Cloud Run across environments (`dev`, `prod`).

## Architecture & Service Connections

The A2A Agent runs as an independent conversational service that:
1. Orchestrates multi-agent reasoning and user chats via **Vertex AI (Gemini 3.7 Flash)**.
2. Invokes tools on the **MCP Server** (`ai-daily-brief-mcp-${environment}`) over authenticated Cloud Run service-to-service IAM (`roles/run.invoker`).

| Component | Dev (`dev.tfvars`) | Prod (`prod.tfvars`) |
| :--- | :--- | :--- |
| **Config Profile (`MODENV_PROFILE`)** | `dev` (`configs/agent/.env.dev.toml`) | `prod` (`configs/agent/.env.prod.toml`) |
| **Cloud Run Service** | `ai-daily-brief-agent-dev` | `ai-daily-brief-agent-prod` |
| **Target MCP Service** | `ai-daily-brief-mcp-dev` | `ai-daily-brief-mcp-prod` |
| **Min Instances** | `0` (scale-to-zero) | `1` (warm instance) |
| **Max Instances** | `5` | `20` |
| **Service Account** | `ai-daily-brief-agent-sa-dev@...` | `ai-daily-brief-agent-sa-prod@...` |
| **IAM Roles** | `roles/aiplatform.user`, `roles/run.invoker` (on MCP) | `roles/aiplatform.user`, `roles/run.invoker` (on MCP) |

---

## Deployment Commands

### 1. Development Environment
```bash
cd deployments/agent/terraform

# Initialize Terraform
terraform init

# Plan Dev Deployment
terraform plan -var-file=environments/dev.tfvars

# Apply Dev Deployment
terraform apply -var-file=environments/dev.tfvars
```

### 2. Production Environment
```bash
cd deployments/agent/terraform

# Plan Prod Deployment
terraform plan -var-file=environments/prod.tfvars

# Apply Prod Deployment
terraform apply -var-file=environments/prod.tfvars
```
