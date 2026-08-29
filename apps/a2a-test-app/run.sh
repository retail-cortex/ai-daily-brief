#!/usr/bin/env bash
set -e

if [ -n "${BUILD_WORKSPACE_DIRECTORY:-}" ]; then
  cd "${BUILD_WORKSPACE_DIRECTORY}/apps/a2a-test-app"
fi

exec uv run python -m a2a_test_app.main
