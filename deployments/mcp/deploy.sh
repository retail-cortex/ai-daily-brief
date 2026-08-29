#!/usr/bin/env bash
# Copyright 2026 Retail Cortex
# Deployment script for AI Daily Brief MCP Server to Google Cloud Run

set -euo pipefail

PROJECT_ID="${GOOGLE_CLOUD_PROJECT:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${GCP_REGION:-us-central1}"
ENVIRONMENT="${MODENV_PROFILE:-dev}"
SERVICE_NAME="ai-daily-brief-mcp-${ENVIRONMENT}"
IMAGE_NAME="${REGION}-docker.pkg.dev/${PROJECT_ID}/ai-daily-brief/${SERVICE_NAME}:latest"

if [[ -z "${PROJECT_ID}" ]]; then
  echo "❌ Error: GOOGLE_CLOUD_PROJECT is not set and no default gcloud project found."
  exit 1
fi

echo "🚀 Building container image via Google Cloud Build..."
gcloud builds submit \
  --project="${PROJECT_ID}" \
  --config="deployments/mcp/cloudbuild.yaml" \
  --substitutions="_IMAGE_URL=${IMAGE_NAME}" \
  .

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
  --max-instances=10 \
  --service-account="ai-daily-brief-sa-${ENVIRONMENT}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --update-env-vars="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},MODENV_PROFILE=${ENVIRONMENT}" \
  --set-secrets="ALLOYDB_DATABASE_URL=ai-daily-brief-db-url-${ENVIRONMENT}:latest"

echo "✅ Deployment completed successfully!"
