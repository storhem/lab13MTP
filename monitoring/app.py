import logging
import os

import httpx
from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(name)s %(levelname)s %(message)s",
)
logger = logging.getLogger("monitoring")

ORCHESTRATOR_URL = os.getenv("ORCHESTRATOR_URL", "http://orchestrator:8000")

app = FastAPI(title="Trading Platform Monitoring", version="1.0.0")
templates = Jinja2Templates(directory="templates")


async def _fetch_json(url: str) -> dict:
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.get(url)
            resp.raise_for_status()
            return resp.json()
    except Exception as e:
        logger.warning(f"Failed to fetch {url}: {e}")
        return {}


@app.get("/", response_class=HTMLResponse)
async def dashboard(request: Request):
    agents_data = await _fetch_json(f"{ORCHESTRATOR_URL}/agents")
    metrics_data = await _fetch_json(f"{ORCHESTRATOR_URL}/metrics")
    health_data = await _fetch_json(f"{ORCHESTRATOR_URL}/health")

    agents = agents_data.get("agents", [])
    orchestrator_status = health_data.get("status", "unknown")
    queue_length = metrics_data.get("queue_length", 0)

    return templates.TemplateResponse(
        "index.html",
        {
            "request": request,
            "agents": agents,
            "orchestrator_status": orchestrator_status,
            "queue_length": queue_length,
            "orchestrator_url": ORCHESTRATOR_URL,
        },
    )


@app.get("/api/status")
async def api_status():
    agents_data = await _fetch_json(f"{ORCHESTRATOR_URL}/agents")
    metrics_data = await _fetch_json(f"{ORCHESTRATOR_URL}/metrics")
    health_data = await _fetch_json(f"{ORCHESTRATOR_URL}/health")
    return {
        "orchestrator": health_data,
        "agents": agents_data.get("agents", []),
        "metrics": metrics_data,
    }
