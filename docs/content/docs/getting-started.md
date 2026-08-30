---
title: "Getting Started"
weight: 1
---

# 🚀 Getting Started

This guide walks you through configuring, compiling, and running the **AI Daily Brief** stack locally using Bazel, `modenv` profile management, and Terraform infrastructure variable definitions.

---

## 📋 Prerequisites

Before running the application, ensure the following tools are installed:

- **Bazel / Bazelisk**: Hermetic build toolchain (`brew install bazelisk`)
- **Python Package Manager (`uv`)**: For running the interactive A2A test harness (`brew install uv` or `curl -LsSf https://astral.sh/uv/install.sh | sh`)
- **Google Cloud SDK (`gcloud`)**: Required for Vertex AI Application Default Credentials (ADC):
  ```bash
  # Authenticate local credentials for Vertex AI Gemini models
  gcloud auth application-default login

  # Set default active project
  gcloud config set project YOUR_GCP_PROJECT_ID
  ```

---

## ⚙️ Step 1: Application Configuration (`.env.toml`)

AI Daily Brief utilizes **`modenv`** for hierarchical configuration and profile resolution across both the **MCP Server** (`configs/mcp/`) and the **A2A Agent** (`configs/agent/`).

### Configuration Precedence Order

For any given environment profile (e.g. `dev`, `prod`, `integration`), `modenv` resolves settings in the following order (highest precedence first):

1. `configs/<service>/.env.<profile>.local.toml` *(Uncommitted local secrets/overrides)*
2. `configs/<service>/.env.<profile>.toml` *(Environment-specific configuration)*
3. `configs/<service>/.env.local.toml` *(Default local uncommitted overrides)*
4. `configs/<service>/.env.toml` *(Default base configuration)*

### Setting Up Local Environment Files

Create your local development configuration files by copying from the provided examples:

```bash
# 1. Configure the MCP Server for development
cp configs/mcp/example-dev.toml configs/mcp/.env.dev.toml

# 2. Configure the A2A Agent for development
cp configs/agent/example-dev.toml configs/agent/.env.dev.toml
```

### Configuration Structure Reference

#### MCP Server (`configs/mcp/.env.dev.toml`)
```toml
[server]
cron_schedule = "0 8 * * 1-5"      # Daily brief scheduled crawl (Monday-Friday 8am)

[database]
dialect = "sqlite"                 # "sqlite" for local dev, "postgres" for AlloyDB
database = "data/ai_daily_brief.db" # Local file path or database name
require_ssl = false                # Set true when connecting to Cloud SQL / AlloyDB

[google_cloud]
project_region = "us-central1"      # Vertex AI inference region
```

#### A2A Agent (`configs/agent/.env.dev.toml`)
```toml
agent_name = "ai-daily-brief-a2a-agent-dev"

[google_cloud]
project_id = "YOUR_GCP_DEV_PROJECT_ID"
project_region = "us-central1"

[mcp_servers.daily_brief]
url = "http://localhost:8080"      # Local MCP endpoint (or Cloud Run URL in staging/prod)
timeout_seconds = 60

[gemini.default]
model = "gemini-3.7-flash"         # Gemini model family
region = "us-central1"
auth_mode = "vertex_adc"           # Vertex AI via Application Default Credentials
```

---

## 🌍 Step 2: Terraform Infrastructure Variables (`*.tfvars`)

When preparing deployments to **Google Cloud Run** and **Google Cloud AlloyDB**, each service has its own dedicated Terraform module under `deployments/`.

### Setting Up Environment Variables

Create `.tfvars` files from the provided templates:

```bash
# 1. MCP Server Terraform Variables
cp deployments/mcp/terraform/environments/dev.tfvars.example deployments/mcp/terraform/environments/dev.tfvars
cp deployments/mcp/terraform/environments/prod.tfvars.example deployments/mcp/terraform/environments/prod.tfvars

# 2. A2A Agent Terraform Variables
cp deployments/agent/terraform/environments/dev.tfvars.example deployments/agent/terraform/environments/dev.tfvars
cp deployments/agent/terraform/environments/prod.tfvars.example deployments/agent/terraform/environments/prod.tfvars
```

### Key Terraform Variables Reference

| Variable | Description | Example / Default |
| :--- | :--- | :--- |
| `project_id` | Target Google Cloud Project ID | `"your-project-id"` |
| `region` | Target GCP deployment region | `"us-central1"` |
| `environment` | Deployment lifecycle (`dev`, `staging`, `prod`) | `"dev"` |
| `service_name` | Cloud Run service name | `"ai-daily-brief-mcp"` |
| `min_instances` | Minimum Cloud Run instances (0 for scale-to-zero) | `0` |
| `max_instances` | Maximum Cloud Run instances | `5` |
| `enable_alloydb` | Provision managed Google Cloud AlloyDB cluster | `true` |
| `alloydb_database_name` | Primary PostgreSQL database name | `"aibrief"` |
| `mcp_service_name` | Target MCP service for A2A Agent binding | `"ai-daily-brief-mcp"` |

---

## 💻 Step 3: Launching the Local Stack

### Parallel Multi-Runner (Recommended)

Start the entire development environment (all 4 services in parallel) with a single command:

```bash
bazel run //:dev
```

This concurrently launches:
1. 📚 **Hugo Documentation Site**: `http://localhost:1313`
2. 🧪 **A2A Diagnostic Test App**: `http://localhost:8888`
3. 🛡️ **MCP Server & REST Control Plane**: `http://localhost:8080` (`MODENV_PROFILE=dev`)
4. 🤖 **Agent-to-Agent (A2A) Service**: `http://localhost:8081` (`MODENV_PROFILE=dev`)

---

### Running Individual Services

You can also run individual services as needed:

```bash
# 1. Run MCP & REST Server on port 8080
bazel run //:run

# 2. Run A2A Agent on port 8081
bazel run //:run-agent

# 3. Run interactive Web Test Harness on port 8888
bazel run //:run-test-app

# 4. Serve the Hugo documentation site on port 1313
bazel run //:serve-docs
```

---

### Activating Specific Environment Profiles

Pass `MODENV_PROFILE` (or `ENV`) to run with specific configurations:

```bash
# Run with integration profile (AlloyDB / Cloud Postgres)
MODENV_PROFILE=integration bazel run //:run

# Run with test profile (In-memory SQLite)
MODENV_PROFILE=test bazel run //:run

# Run with dev profile explicitly
MODENV_PROFILE=dev bazel run //:run
```

---

## 🔍 Step 4: Verification & Diagnostic Playground

### 1. Service Health Probes
```bash
# Verify MCP Server health
curl http://localhost:8080/healthz

# Verify A2A Agent health
curl http://localhost:8081/healthz
```

### 2. Interactive Web Test Harness
Open `http://localhost:8888` in your browser to access the test UI. This visual harness provides:
- Live status checks for the MCP Server and A2A Agent
- Form-based invocation of MCP tools (`generate_brief`, `search_briefs`, `get_stats`)
- Live preview of generated **A2UI Adaptive Card** HTML/JSON payloads
- A2A multi-turn chat stream with execution telemetry

---

## 🛠️ CLI Flags Reference

### MCP Server (`cmd/mcp-server`)
```text
Usage of mcp-server:
  -port string
        Port to listen on (defaults to $PORT or 8080 for Cloud Run)
  -db string
        Path to SQLite database file or AlloyDB/PostgreSQL connection string
  -dsn string
        Explicit Google Cloud AlloyDB / PostgreSQL DSN
  -stdio
        Run in stdio mode for local desktop MCP clients (Antigravity/Claude/Cursor)
```

### A2A Agent (`cmd/a2a-agent`)
```text
Usage of a2a-agent:
  -port string
        Port to listen on (defaults to $PORT or 8081 for Cloud Run)
  -in-process
        Force direct in-process ToolExecutor mode backed directly by GORM and AlloyDB
  -db string
        Path to SQLite database file or AlloyDB/PostgreSQL connection string for in-process mode
  -dsn string
        Explicit Google Cloud AlloyDB / PostgreSQL DSN for direct in-process execution
  -mcp-url string
        Target remote MCP Server endpoint URL for remote tool dispatch fallback
```
