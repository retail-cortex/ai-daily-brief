#!/usr/bin/env bash
# Copyright 2026 Retail Cortex
# Deployment script for AI Daily Brief A2A Agent to Google Cloud Run

set -euo pipefail

PROJECT_ID="${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${GCP_REGION:-us-central1}"
ENVIRONMENT="${MODENV_PROFILE:-dev}"
SERVICE_NAME="ai-daily-brief-agent-${ENVIRONMENT}"
IMAGE_NAME="${REGION}-docker.pkg.dev/${PROJECT_ID}/ai-daily-brief/ai-daily-brief-agent:latest"
MCP_SERVICE_NAME="ai-daily-brief-mcp-${ENVIRONMENT}"

if [[ -z "${PROJECT_ID}" ]]; then
  echo "❌ Error: GOOGLE_CLOUD_PROJECT is not set and no default gcloud project found."
  exit 1
fi

echo "🚀 Building A2A Agent container image via Google Cloud Build..."
gcloud builds submit \
  --project="${PROJECT_ID}" \
  --config="deployments/agent/cloudbuild.yaml" \
  --substitutions="_IMAGE_URL=${IMAGE_NAME}" \
  .

# Discover MCP service URL if not explicitly provided
MCP_URL="${MCP_SERVER_URL:-$(gcloud run services describe "${MCP_SERVICE_NAME}" --project="${PROJECT_ID}" --region="${REGION}" --format='value(status.url)' 2>/dev/null || echo "")}"

echo "☁️ Deploying ${SERVICE_NAME} to Google Cloud Run (${REGION})..."
gcloud run deploy "${SERVICE_NAME}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --image="${IMAGE_NAME}" \
  --platform="managed" \
  --no-allow-unauthenticated \
  --port=8080 \
  --memory="1Gi" \
  --cpu="1" \
  --min-instances=0 \
  --max-instances=5 \
  --service-account="ai-daily-brief-agent-sa-${ENVIRONMENT}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --set-env-vars="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},MODENV_PROFILE=${ENVIRONMENT},MCP_SERVER_URL=${MCP_URL}"

echo "✅ A2A Agent deployment completed successfully!"
