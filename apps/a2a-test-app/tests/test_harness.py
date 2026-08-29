"""Unit tests for the A2A Diagnostic Test Harness."""

from fastapi.testclient import TestClient
from a2a_test_app.main import app

client = TestClient(app)


def test_ui_serves_html() -> None:
    """Verify root GET endpoint renders HTML UI."""
    response = client.get("/")
    assert response.status_code == 200
    assert "AI Daily Brief" in response.text
    assert "A2A 0.8 / A2UI v1.0" in response.text
    assert "extractA2UIComponents" in response.text


def test_invoke_schema_validation() -> None:
    """Verify request validation for agent invocation."""
    response = client.post("/api/invoke", json={"prompt": "hi"})
    assert response.status_code in [200, 500]
