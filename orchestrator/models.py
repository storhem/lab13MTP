from pydantic import BaseModel
from typing import List, Optional


class TradeRequest(BaseModel):
    symbols: List[str]


class QuoteResult(BaseModel):
    task_id: str
    quotes: list
    status: str


class MarketResult(BaseModel):
    task_id: str
    avg_price: float
    trend: str
    total_volume: int
    recommendation: str
    status: str


class RiskResult(BaseModel):
    task_id: str
    risk_level: str
    risk_score: float
    approved: bool
    status: str


class TradeResult(BaseModel):
    task_id: str
    order_id: Optional[str] = None
    status: str
    execution_price: Optional[float] = None
    message: str


class LLMResult(BaseModel):
    task_id: str
    analysis: str
    status: str


class PipelineResult(BaseModel):
    task_id: str
    symbols: List[str]
    quotes: list
    market_analysis: dict
    risk_assessment: dict
    trade_execution: dict
    llm_analysis: str
    final_status: str
