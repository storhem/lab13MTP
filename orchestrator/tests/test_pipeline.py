import uuid
from unittest.mock import AsyncMock, MagicMock

import pytest

# Ensure the orchestrator package directory is on sys.path when running from project root.
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))


@pytest.mark.asyncio
async def test_pipeline_execute_success():
    from pipeline import TradingPipeline

    mock_orchestrator = MagicMock()
    task_id = str(uuid.uuid4())

    mock_orchestrator.send_task_with_retry = AsyncMock(
        side_effect=[
            # quotes
            {
                "task_id": task_id,
                "quotes": [{"symbol": "AAPL", "price": 150.0, "volume": 60000}],
                "status": "ok",
            },
            # market
            {
                "task_id": task_id,
                "avg_price": 150.0,
                "trend": "bullish",
                "total_volume": 60000,
                "recommendation": "BUY",
                "status": "ok",
            },
            # risk
            {
                "task_id": task_id,
                "risk_level": "LOW",
                "risk_score": 0.2,
                "approved": True,
                "status": "ok",
            },
            # trade
            {
                "task_id": task_id,
                "order_id": "ord-1",
                "status": "EXECUTED",
                "execution_price": 150.5,
                "message": "ok",
            },
            # llm
            {
                "task_id": task_id,
                "analysis": "Сделка успешно исполнена.",
                "status": "ok",
            },
        ]
    )

    pipeline = TradingPipeline(mock_orchestrator)
    result = await pipeline.execute(["AAPL"])

    assert result["final_status"] == "EXECUTED"
    assert result["symbols"] == ["AAPL"]
    assert "market_analysis" in result
    assert result["market_analysis"]["recommendation"] == "BUY"
    assert result["llm_analysis"] == "Сделка успешно исполнена."


@pytest.mark.asyncio
async def test_pipeline_rejected_trade():
    from pipeline import TradingPipeline

    mock_orchestrator = MagicMock()
    task_id = str(uuid.uuid4())

    mock_orchestrator.send_task_with_retry = AsyncMock(
        side_effect=[
            {
                "task_id": task_id,
                "quotes": [{"symbol": "TSLA", "price": 900.0, "volume": 2000}],
                "status": "ok",
            },
            {
                "task_id": task_id,
                "avg_price": 900.0,
                "trend": "bullish",
                "total_volume": 2000,
                "recommendation": "BUY",
                "status": "ok",
            },
            {
                "task_id": task_id,
                "risk_level": "HIGH",
                "risk_score": 0.9,
                "approved": False,
                "status": "ok",
            },
            {
                "task_id": task_id,
                "order_id": None,
                "status": "REJECTED",
                "execution_price": None,
                "message": "Trade rejected due to risk assessment",
            },
            {
                "task_id": task_id,
                "analysis": "Сделка отклонена из-за высокого риска.",
                "status": "ok",
            },
        ]
    )

    pipeline = TradingPipeline(mock_orchestrator)
    result = await pipeline.execute(["TSLA"])

    assert result["final_status"] == "REJECTED"
    assert result["risk_assessment"]["approved"] is False
