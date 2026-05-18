import asyncio
import logging
import os
from contextlib import asynccontextmanager
from typing import List

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from auction import AuctionManager
from orchestrator import Orchestrator
from pipeline import TradingPipeline
from scaler import DynamicScaler

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(name)s %(levelname)s %(message)s",
)

NATS_URL = os.getenv("NATS_URL", "nats://nats:4222")

orchestrator = Orchestrator(nats_url=NATS_URL)
pipeline: TradingPipeline = None
auction_manager: AuctionManager = None
scaler: DynamicScaler = None
_scaler_task: asyncio.Task = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global pipeline, auction_manager, scaler, _scaler_task

    await orchestrator.connect()
    await orchestrator.setup_subscriptions()

    pipeline = TradingPipeline(orchestrator)
    auction_manager = AuctionManager(orchestrator.nc)
    scaler = DynamicScaler()
    _scaler_task = asyncio.create_task(scaler.run())

    yield

    scaler.stop()
    if _scaler_task:
        _scaler_task.cancel()
        try:
            await _scaler_task
        except asyncio.CancelledError:
            pass

    await orchestrator.close()


app = FastAPI(
    title="Trading Platform Orchestrator",
    description="Orchestrator for the multi-agent trading platform",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class TradeRequest(BaseModel):
    symbols: List[str]


@app.get("/health")
async def health():
    return {"status": "healthy", "service": "orchestrator"}


@app.post("/trade")
async def execute_trade(request: TradeRequest):
    if not request.symbols:
        raise HTTPException(status_code=400, detail="symbols list is empty")
    try:
        result = await pipeline.execute(request.symbols)
        return result
    except TimeoutError as e:
        raise HTTPException(status_code=504, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/agents")
async def list_agents():
    return {
        "agents": [
            {"name": "quotes-agent", "type": "go", "topic": "trading.quotes.collect"},
            {"name": "market-analyzer", "type": "go", "topic": "trading.market.analyze"},
            {"name": "risk-manager", "type": "go", "topic": "trading.risk.assess"},
            {"name": "trade-executor", "type": "go", "topic": "trading.trade.execute"},
            {"name": "llm-analyst", "type": "python", "topic": "trading.llm.analyze"},
        ]
    }


@app.get("/metrics")
async def metrics():
    queue_length = 0
    if scaler:
        queue_length = await scaler.get_queue_length()
    return {
        "service": "orchestrator",
        "pipeline_requests": "see POST /trade",
        "queue_length": queue_length,
    }


@app.get("/auction/demo")
async def auction_demo():
    if not auction_manager:
        raise HTTPException(status_code=503, detail="Auction manager not initialized")
    result = await auction_manager.demo_auction()
    return result
