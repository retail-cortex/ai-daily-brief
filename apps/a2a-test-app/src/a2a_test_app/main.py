"""Interactive Web Test Harness for A2A Agent and MCP Server.

Validates Gemini Enterprise / A2A Protocol compliance and renders A2UI components.
"""

import os
import subprocess
import time

import httpx
from fastapi import FastAPI
from fastapi.responses import HTMLResponse, JSONResponse
from pydantic import BaseModel, Field

DEFAULT_AGENT_URL: str = os.getenv(
    "A2A_AGENT_URL",
    "http://localhost:8081"
)
DEFAULT_MCP_URL: str = os.getenv(
    "MCP_SERVER_URL",
    "http://localhost:8080"
)
DEFAULT_SA_IMPERSONATE: str = os.getenv(
    "A2A_IMPERSONATE_SA",
    ""
)

app = FastAPI(
    title="AI Daily Brief A2A & MCP Test Harness",
    version="1.0.0",
    description="Interactive diagnostic and execution playground for Google Gemini Enterprise A2A Agent"
)


class InvokeRequest(BaseModel):
    prompt: str = Field(..., description="Task prompt or user utterance")
    context_id: str | None = Field(default=None, description="A2A context ID")
    method: str | None = Field(default="message/send", description="A2A JSON-RPC Method")
    agent_url: str | None = Field(default=None, description="Target A2A Agent URL")


class MCPToolRequest(BaseModel):
    tool_name: str = Field(..., description="MCP Tool Name")
    arguments: dict[str, str | int | bool | float] = Field(default_factory=dict, description="Tool Arguments")
    mcp_url: str | None = Field(default=None, description="Target MCP Server URL")


def get_gcp_auth_token(target_url: str = "") -> str:
    """Retrieve Google Cloud identity token with audience and impersonation support."""
    # 1. Try audience token via service account impersonation
    if target_url and DEFAULT_SA_IMPERSONATE:
        try:
            aud = target_url.rstrip("/")
            if "/agent" in aud:
                aud = aud.split("/agent")[0]
            if "/api" in aud:
                aud = aud.split("/api")[0]
            cmd = [
                "gcloud", "auth", "print-identity-token",
                f"--impersonate-service-account={DEFAULT_SA_IMPERSONATE}",
                "--include-email",
                f"--audiences={aud}"
            ]
            res = subprocess.run(cmd, capture_output=True, text=True, check=True, timeout=10)
            token = res.stdout.strip()
            if token:
                return token
        except (subprocess.SubprocessError, OSError):
            pass

    # 2. Try direct identity token
    try:
        res = subprocess.run(
            ["gcloud", "auth", "print-identity-token"],
            capture_output=True,
            text=True,
            check=True,
            timeout=10,
        )
        token = res.stdout.strip()
        if token:
            return token
    except (subprocess.SubprocessError, OSError):
        pass

    # 3. Try access token fallback
    try:
        res = subprocess.run(
            ["gcloud", "auth", "print-access-token"],
            capture_output=True,
            text=True,
            check=True,
            timeout=10,
        )
        return res.stdout.strip()
    except (subprocess.SubprocessError, OSError) as err:
        print(f"[Auth Notice] Could not acquire GCP token: {err}")
        return ""


from pathlib import Path

INDEX_HTML_PATH = Path(__file__).parent / "index.html"


@app.get("/", response_class=HTMLResponse)
async def serve_ui() -> HTMLResponse:
    """Renders the comprehensive diagnostic test harness web UI."""
    content = INDEX_HTML_PATH.read_text(encoding="utf-8")
    return HTMLResponse(content=content)



@app.post("/api/invoke")
async def invoke_agent(req: InvokeRequest) -> JSONResponse:
    """Proxy invocation with Google Cloud authentication and JSON-RPC compliance."""
    target_url = req.agent_url or DEFAULT_AGENT_URL
    if not target_url.endswith("/agent/invoke") and not target_url.endswith("/a2a") and not target_url.endswith("/mcp"):
        invoke_url = f"{target_url.rstrip('/')}/agent/invoke"
    else:
        invoke_url = target_url

    token = get_gcp_auth_token(target_url)

    headers = {
        "Content-Type": "application/json",
        "Accept": "application/json",
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"

    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": req.method or "message/send",
        "params": {
            "message": {
                "contextId": req.context_id or f"ctx-{int(time.time())}",
                "role": "user",
                "parts": [
                    {"kind": "text", "text": req.prompt}
                ]
            }
        }
    }

    async with httpx.AsyncClient(timeout=45.0) as client:
        try:
            resp = await client.post(invoke_url, json=payload, headers=headers)
            return JSONResponse(status_code=resp.status_code, content=resp.json())
        except (httpx.HTTPError, ValueError, OSError) as err:
            return JSONResponse(status_code=500, content={"error": str(err)})


@app.get("/api/health")
async def health_check(agent_url: str | None = None) -> JSONResponse:
    """Checks the agent-card.json endpoint on the agent."""
    target_url = agent_url or DEFAULT_AGENT_URL
    card_url = f"{target_url.rstrip('/')}/.well-known/agent-card.json"
    token = get_gcp_auth_token(target_url)

    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    async with httpx.AsyncClient(timeout=10.0) as client:
        try:
            resp = await client.get(card_url, headers=headers)
            return JSONResponse(status_code=200, content={"status": resp.status_code, "card": resp.json()})
        except (httpx.HTTPError, ValueError, OSError) as err:
            return JSONResponse(status_code=500, content={"status": 500, "error": str(err)})


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8888"))
    uvicorn.run(app, host="0.0.0.0", port=port)
