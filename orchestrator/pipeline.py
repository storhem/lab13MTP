import logging
import uuid

from orchestrator import Orchestrator


class TradingPipeline:
    def __init__(self, orchestrator: Orchestrator):
        self.orchestrator = orchestrator
        self.logger = logging.getLogger("pipeline")

    async def execute(self, symbols: list) -> dict:
        task_id = str(uuid.uuid4())
        self.logger.info(f"Starting pipeline {task_id} for symbols: {symbols}")

        # Step 1: Collect quotes
        quotes_result = await self.orchestrator.send_task_with_retry(
            "trading.quotes.collect",
            {"task_id": task_id, "symbols": symbols},
        )
        self.logger.info(f"[{task_id}] Quotes collected: {len(quotes_result.get('quotes', []))} entries")

        # Step 2: Analyse market
        market_result = await self.orchestrator.send_task_with_retry(
            "trading.market.analyze",
            {"task_id": task_id, "quotes": quotes_result.get("quotes", [])},
        )
        self.logger.info(
            f"[{task_id}] Market analysed: trend={market_result.get('trend')} "
            f"recommendation={market_result.get('recommendation')}"
        )

        # Step 3: Assess risk
        risk_result = await self.orchestrator.send_task_with_retry(
            "trading.risk.assess",
            {
                "task_id": task_id,
                "recommendation": market_result.get("recommendation"),
                "avg_price": market_result.get("avg_price"),
                "total_volume": market_result.get("total_volume"),
            },
        )
        self.logger.info(
            f"[{task_id}] Risk assessed: level={risk_result.get('risk_level')} "
            f"approved={risk_result.get('approved')}"
        )

        # Step 4: Execute trade
        trade_result = await self.orchestrator.send_task_with_retry(
            "trading.trade.execute",
            {
                "task_id": task_id,
                "recommendation": market_result.get("recommendation"),
                "approved": risk_result.get("approved"),
                "risk_level": risk_result.get("risk_level"),
                "symbols": symbols,
            },
        )
        self.logger.info(f"[{task_id}] Trade status: {trade_result.get('status')}")

        # Step 5: LLM analysis
        llm_result = await self.orchestrator.send_task_with_retry(
            "trading.llm.analyze",
            {
                "task_id": task_id,
                "symbols": symbols,
                "trend": market_result.get("trend"),
                "recommendation": market_result.get("recommendation"),
                "risk_level": risk_result.get("risk_level"),
                "trade_status": trade_result.get("status"),
                "execution_price": trade_result.get("execution_price"),
            },
        )
        self.logger.info(f"[{task_id}] LLM analysis done")

        return {
            "task_id": task_id,
            "symbols": symbols,
            "quotes": quotes_result.get("quotes", []),
            "market_analysis": market_result,
            "risk_assessment": risk_result,
            "trade_execution": trade_result,
            "llm_analysis": llm_result.get("analysis", ""),
            "final_status": trade_result.get("status", "UNKNOWN"),
        }
