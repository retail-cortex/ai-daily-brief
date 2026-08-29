---
title: "Troubleshooting Guide"
weight: 9
---

# 🛠️ Troubleshooting Guide

This guide covers common issues, edge cases, and debugging workflows when developing, deploying, and operating **AI Daily Brief** on Google Cloud Run, AlloyDB, and Gemini Enterprise.

---

## 1. Connecting to AlloyDB with Local Auth Proxy & Running MCP Locally

When developing or debugging locally, you can connect your local workstation directly to the private Google Cloud AlloyDB cluster using the **AlloyDB Auth Proxy**.

```
┌─────────────────────────────────┐          ┌───────────────────────────┐          ┌───────────────────────────┐
│       Local Workstation         │          │     Google Cloud IAM      │          │     Private VPC Network   │
│                                 │          │                           │          │                           │
│  [Local MCP Server]             │          │                           │          │                           │
│   (port :8080)                  │          │                           │          │                           │
│        │                        │          │                           │          │                           │
│        ▼ localhost:5432         │          │                           │          │                           │
│  [alloydb-auth-proxy] ──────────┼─────────►│ Authenticates via gcloud  │─────────►│  AlloyDB Primary Instance │
│   (mTLS Tunnel)                 │          │ (roles/alloydb.client)    │          │  (Port 5432, Private IP)  │
└─────────────────────────────────┘          └───────────────────────────┘          └───────────────────────────┘
```

### Step 1: Install the AlloyDB Auth Proxy
If you don't already have the proxy installed:

```bash
# macOS (Homebrew or direct binary)
curl -o alloydb-auth-proxy https://storage.googleapis.com/alloydb-auth-proxy/v1.10.0/alloydb-auth-proxy.darwin.arm64
chmod +x alloydb-auth-proxy
sudo mv alloydb-auth-proxy /usr/local/bin/

# Linux AMD64
curl -o alloydb-auth-proxy https://storage.googleapis.com/alloydb-auth-proxy/v1.10.0/alloydb-auth-proxy.linux.amd64
chmod +x alloydb-auth-proxy
sudo mv alloydb-auth-proxy /usr/local/bin/
```

### Step 2: Grant Required IAM Roles
Ensure your user identity has the `AlloyDB Client` role:

```bash
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="user:$(gcloud config get-value account)" \
  --role="roles/alloydb.client"
```

### Step 3: Start the AlloyDB Auth Proxy
Run the proxy locally to create an encrypted mTLS tunnel to your AlloyDB instance:

```bash
# Instance connection string format:
# projects/<PROJECT_ID>/locations/<REGION>/clusters/<CLUSTER_NAME>/instances/<INSTANCE_NAME>

alloydb-auth-proxy \
  "projects/${PROJECT_ID}/locations/us-central1/clusters/ai-daily-brief-alloydb-dev/instances/ai-daily-brief-alloydb-dev-primary" \
  --port=5432
```

> [!NOTE]
> Leave this terminal process running in the background. It listens on `127.0.0.1:5432` and forwards traffic securely to AlloyDB.

### Step 4: Verify Database Access with `psql`
In a separate terminal, test the connection using the standard PostgreSQL CLI:

```bash
# Retrieve the database password from Secret Manager (if stored)
DB_PASS=$(gcloud secrets versions access latest --secret="ai-daily-brief-db-password-dev" --project="${PROJECT_ID}" 2>/dev/null || echo "your-db-password")

# Connect via localhost:5432 through the proxy
PGPASSWORD="${DB_PASS}" psql -h 127.0.0.1 -p 5432 -U postgres -d ai_daily_brief -c "\dt"
```

### Step 5: Run the MCP Server Locally Pointing to AlloyDB
Start the local MCP server with `ALLOYDB_DATABASE_URL` pointing at the local proxy port:

```bash
export ALLOYDB_DATABASE_URL="postgres://postgres:${DB_PASS}@127.0.0.1:5432/ai_daily_brief?sslmode=disable"
export GOOGLE_CLOUD_PROJECT="${PROJECT_ID}"
export GCP_REGION="us-central1"
export MODENV_PROFILE="dev"

# Run via Bazel
bazel run //cmd/mcp-server

# Or run directly with Go
go run ./cmd/mcp-server
```

### Step 6: Test Local MCP Server Tools
Verify that your local MCP server is talking to AlloyDB and serving tools with `application/a2ui+json`:

```bash
# 1. Inspect status
curl -s http://localhost:8080/api/status | jq .

# 2. Call MCP list_articles tool
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "list_articles", "arguments": {"limit": 3}}, "id": 1}' | jq .
```

---

## 2. Cloud Run Endpoint Invocations & Probes

### A. `404 (Not Found)` on `GET /healthz`
**Symptom:**
```text
jq: parse error: Invalid numeric literal at line 1, column 10
<!DOCTYPE html>
<title>Error 404 (Not Found)!!1</title>
The requested URL /healthz was not found on this server. That's all we know.
```

**Root Causes:**
1. **Internal Probe Reservation:** Google Front End (GFE) intercepts `/healthz` internally on `.run.app` URLs for container startup and liveness probes. External HTTP requests to `/healthz` over public hostnames are dropped/intercepted before reaching the container.
2. **Audience Mismatch:** If an OIDC identity token is minted with a generic Google Client ID rather than the exact Cloud Run service URL, GFE rejects the request with a standard Google 404.

**Resolution:**
To verify the service externally, invoke the MCP endpoint (`POST /mcp`) or REST API (`GET /api/status`, `POST /api/batch/run`) using an audience-bound OIDC token:

```bash
export ENDPOINT="https://ai-daily-brief-mcp-dev-${PROJECT_NUMBER}.us-central1.run.app"
ACCESS_TOKEN=$(gcloud auth print-access-token)

# Mint OIDC Token bound to the exact Cloud Run audience
TOKEN=$(curl -s -X POST \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"audience\": \"${ENDPOINT}\", \"includeEmail\": true}" \
  "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/ai-daily-brief-sa-dev@${PROJECT_ID}.iam.gserviceaccount.com:generateIdToken" | jq -r .token)

# Test MCP Tool Listing
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}' \
  "${ENDPOINT}/mcp" | jq .
```

---

### B. `FAILED_PRECONDITION: Domain Restricted Sharing (DRS)`
**Symptom:**
```text
ERROR: (gcloud.run.services.add-iam-policy-binding) FAILED_PRECONDITION: One or more users named in the policy do not belong to a permitted customer, perhaps due to an organization policy.
```

**Root Cause:**
Your Google Cloud Organization enforces `constraints/iam.allowedPolicyMemberDomains`, preventing `allUsers` (public unauthenticated access) from being granted `roles/run.invoker`.

**Resolution:**
1. Keep `allow_unauthenticated = false` in your configuration (`deployments/mcp/terraform/environments/dev.tfvars` or `deployments/agent/terraform/environments/dev.tfvars`).
2. Grant IAM Invoker access to specific domain users or runtime service accounts:
   ```bash
   gcloud run services add-iam-policy-binding ai-daily-brief-mcp-dev \
     --region us-central1 \
     --member="serviceAccount:ai-daily-brief-sa-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
     --role="roles/run.invoker"
   ```

---

### C. `403 Forbidden` (`insufficient_scope` / `lacks run.routes.invoke`)
**Symptom:**
```text
The request was not authenticated. The IAM principal lacks {run.routes.invoke} permission.
```

**Resolution:**
Ensure the caller has the required IAM roles:
```bash
# Grant project-level invoker role
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:YOUR_EMAIL" \
  --role="roles/run.invoker"

# Grant Service Account Token & OpenID Token Creator roles (to mint audience tokens)
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:YOUR_EMAIL" \
  --role="roles/iam.serviceAccountOpenIdTokenCreator"

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:YOUR_EMAIL" \
  --role="roles/iam.serviceAccountTokenCreator"
```

---

## 3. Database Persistence & Redeployment Safety

### Will the Database be Recreated on Application Redeployment?
**No, the database is completely preserved across all container redeployments and Terraform updates.**

1. **Decoupled Lifecycle:** In `deployments/mcp/terraform/main.tf`, `google_alloydb_cluster` and `google_alloydb_instance` are standalone resources decoupled from `google_cloud_run_v2_service`. Redeploying the container image (`gcloud run deploy` or Cloud Build) updates only the stateless Cloud Run container revision.
2. **Non-Destructive Migrations:** On container startup, `internal/database/db.go` runs GORM's `AutoMigrate`:
   - It executes `CREATE TABLE IF NOT EXISTS` and `ADD COLUMN` queries.
   - It **never** drops existing tables, indexes, or rows.
3. **Primary Key Deduplication:** Crawled items use deterministic composite IDs (`gcp-rel-<hash>`, `arxiv-<hash>`). Subsequent crawl runs safely skip existing records (`ON CONFLICT DO NOTHING`).

---

## 4. Gemini Enterprise & A2UI MIME Types

### Standard MIME Type: `application/a2ui+json`
When integrating the MCP server with Gemini Enterprise, tool call outputs must return typed content blocks rather than generic strings.

The MCP handler returns:
- **Block 1 (`type: "text"`)**: Human-readable ASCII / Unicode cards.
- **Block 2 (`type: "resource"`)**: Embedded resource with **`mimeType: "application/a2ui+json"`** containing the serialized A2UI card component tree:

```json
{
  "type": "resource",
  "resource": {
    "uri": "brief://a2ui/articles/list",
    "mimeType": "application/a2ui+json",
    "text": "[{\"id\":\"item_1\",\"type\":\"article_card\",\"title\":\"...\"}]"
  }
}
```

> [!TIP]
> For complete details on registering the Agent Card and configuring IAM permissions for Discovery Engine, see the [Gemini Enterprise Integration Guide](/docs/gemini_enterprise/).

---

## 5. Triggering the Batch Crawl Process

### Method 1: Via REST API
```bash
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  "${ENDPOINT}/api/batch/run" | jq .
```

### Method 2: Via Model Context Protocol (MCP)
```bash
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "trigger_crawl", "arguments": {}}, "id": 1}' \
  "${ENDPOINT}/mcp" | jq .
```

### Method 3: Automated Cron Scheduler
The MCP server includes a built-in background cron scheduler (`robfig/cron/v3`).
- **Default Schedule:** Daily at 08:00 UTC (`0 8 * * *`).
- **Configuration:** Set `crawler.schedule` in `configs/mcp/.env.toml` or via environment variable.

---

## 6. Verifying Database Logs & Execution History

### Inspect Cloud Run Application Logs
```bash
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="ai-daily-brief-mcp-dev"' \
  --limit=25 \
  --format="table(timestamp,textPayload)"
```

### Retrieve Execution Run History
```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  "${ENDPOINT}/api/runs" | jq .
```
