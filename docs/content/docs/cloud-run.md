---
title: "Cloud Run Deployment"
weight: 6
---

# ☁️ Google Cloud Run Deployment

AI Daily Brief provides standalone, multi-stage container configurations and Terraform stacks for both the **MCP Server** and the **A2A Agent**.

---

## 1. Directory Structure

```text
deployments/
├── agent/                         # A2A Agent Service Deployment
│   ├── Dockerfile
│   ├── cloudbuild.yaml
│   ├── deploy.sh
│   └── terraform/                 # Agent Cloud Run + Vertex AI IAM
│       ├── main.tf
│       ├── variables.tf
│       ├── outputs.tf
│       └── environments/
│           ├── dev.tfvars
│           └── prod.tfvars
└── mcp/                           # MCP Server Service Deployment
    ├── Dockerfile
    ├── cloudbuild.yaml
    ├── deploy.sh
    └── terraform/                 # MCP Cloud Run + AlloyDB + VPC + Secret Manager
        ├── main.tf
        ├── variables.tf
        ├── outputs.tf
        └── environments/
            ├── dev.tfvars
            └── prod.tfvars
```

---

## 2. Container Building via Bazel

Compile Linux ARM64 or AMD64 container binaries directly using Bazel:

```bash
# Build Linux AMD64 Cloud Run binaries
bazel build //cmd/mcp-server:mcp_server_linux_amd64
bazel build //cmd/a2a-agent:a2a_agent_linux_amd64

# Build Linux ARM64 Cloud Run binaries
bazel build //cmd/mcp-server:mcp_server_linux_arm64
bazel build //cmd/a2a-agent:a2a_agent_linux_arm64
```

---

## 3. Deploying the MCP Server

### Via Deployment Script:
```bash
./deployments/mcp/deploy.sh
```

### Via Terraform:
```bash
cd deployments/mcp/terraform
terraform init
terraform apply -var-file=environments/dev.tfvars
```

---

## 4. Deploying the A2A Agent

### Via Deployment Script:
```bash
./deployments/agent/deploy.sh
```

### Via Terraform:
```bash
cd deployments/agent/terraform
terraform init
terraform apply -var-file=environments/dev.tfvars
```

---

## 5. Container Probes & Ingress

- **MCP Server Probes**: Startup probe on TCP `:8080`, Liveness on HTTP `/healthz`.
- **A2A Agent Probes**: Startup probe on TCP `:8080`, Liveness on HTTP `/healthz`.
- **Service-to-Service Auth**: The Agent authenticates to the MCP server using its dedicated Cloud Run service account (`ai-daily-brief-agent-sa-${env}`) bound to `roles/run.invoker`.
