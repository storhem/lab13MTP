import asyncio
import json
import logging
import os
from typing import Optional

import nats
from nats.aio.client import Client as NATS

from gemini_client import GeminiClient
from tracer import init_tracer

logger = logging.getLogger("llm-analyst")


class LLMAnalystAgent:
    def __init__(self):
        self.nats_url: str = os.getenv("NATS_URL", "nats://localhost:4222")
        self.nc: Optional[NATS] = None
        self.gemini = GeminiClient()
        self.tracer = init_tracer("llm-analyst")
        self._processed = 0

    async def connect(self):
        self.nc = await nats.connect(self.nats_url)
        logger.info(f"Connected to NATS at {self.nats_url}")

    async def close(self):
        if self.nc:
            await self.nc.drain()
            logger.info("NATS connection closed")

    async def run(self):
        await self.connect()

        async def message_handler(msg):
            await self._handle_message(msg)

        sub = await self.nc.subscribe("trading.llm.analyze", cb=message_handler)
        logger.info("LLMAnalystAgent listening on trading.llm.analyze")

        try:
            # Run indefinitely until cancelled
            while True:
                await asyncio.sleep(1)
        except asyncio.CancelledError:
            pass
        finally:
            await sub.unsubscribe()
            await self.close()
            logger.info("LLMAnalystAgent stopped")

    async def _handle_message(self, msg):
        with self.tracer.start_as_current_span("llm-analyst.handle") as span:
            try:
                data = json.loads(msg.data.decode())
                task_id = data.get("task_id", "unknown")
                logger.info(f"Received llm.analyze task_id={task_id}")

                span.set_attribute("task_id", task_id)

                trade_summary = {
                    "symbols": data.get("symbols", []),
                    "trend": data.get("trend", "unknown"),
                    "recommendation": data.get("recommendation", "HOLD"),
                    "risk_level": data.get("risk_level", "MEDIUM"),
                    "trade_status": data.get("trade_status", "unknown"),
                    "execution_price": data.get("execution_price", 0),
                }

                analysis_text = await self.gemini.analyze(trade_summary)

                result = {
                    "task_id": task_id,
                    "analysis": analysis_text,
                    "status": "ok",
                }

                await self.nc.publish("trading.llm.done", json.dumps(result).encode())
                self._processed += 1
                logger.info(f"Published llm.done task_id={task_id} processed={self._processed}")

            except Exception as e:
                logger.error(f"Error handling llm.analyze message: {e}")
                # Try to send error response if task_id is known
                try:
                    task_id = json.loads(msg.data.decode()).get("task_id", "unknown")
                    error_result = {
                        "task_id": task_id,
                        "analysis": f"Ошибка анализа: {str(e)}",
                        "status": "error",
                    }
                    await self.nc.publish("trading.llm.done", json.dumps(error_result).encode())
                except Exception:
                    pass
