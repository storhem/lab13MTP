import asyncio
import json
import logging
import uuid
from typing import Dict, Optional

import nats


class Orchestrator:
    def __init__(self, nats_url: str = "nats://localhost:4222"):
        self.nats_url = nats_url
        self.nc: Optional[nats.NATS] = None
        self.pending: Dict[str, asyncio.Future] = {}
        self._subscriptions = []
        self.logger = logging.getLogger("orchestrator")

    async def connect(self, max_retries: int = 10, retry_delay: float = 2.0):
        for attempt in range(max_retries):
            try:
                self.nc = await nats.connect(self.nats_url)
                self.logger.info(f"Connected to NATS: {self.nats_url}")
                return
            except Exception as exc:
                if attempt < max_retries - 1:
                    self.logger.warning(
                        f"NATS connection attempt {attempt + 1}/{max_retries} failed: {exc}. "
                        f"Retrying in {retry_delay}s..."
                    )
                    await asyncio.sleep(retry_delay)
                else:
                    raise RuntimeError(f"Failed to connect to NATS after {max_retries} attempts") from exc

    async def close(self):
        if self.nc:
            await self.nc.drain()

    async def _subscribe_result(self, topic: str):
        async def handler(msg):
            data = json.loads(msg.data.decode())
            task_id = data.get("task_id")
            if task_id and task_id in self.pending:
                if not self.pending[task_id].done():
                    self.pending[task_id].set_result(data)

        sub = await self.nc.subscribe(topic, cb=handler)
        self._subscriptions.append(sub)
        return sub

    async def setup_subscriptions(self):
        await self._subscribe_result("trading.quotes.done")
        await self._subscribe_result("trading.market.done")
        await self._subscribe_result("trading.risk.done")
        await self._subscribe_result("trading.trade.done")
        await self._subscribe_result("trading.llm.done")
        self.logger.info("All result subscriptions are active")

    async def send_task(self, topic: str, payload: dict, timeout: int = 30) -> dict:
        task_id = payload.get("task_id", str(uuid.uuid4()))
        payload["task_id"] = task_id

        loop = asyncio.get_event_loop()
        future = loop.create_future()
        self.pending[task_id] = future

        await self.nc.publish(topic, json.dumps(payload).encode())
        self.logger.info(f"Sent task {task_id} to {topic}")

        try:
            result = await asyncio.wait_for(future, timeout=timeout)
            self.logger.info(f"Got result for task {task_id}")
            return result
        except asyncio.TimeoutError:
            self.logger.error(f"Task {task_id} timed out after {timeout}s")
            raise TimeoutError(f"Task {task_id} timed out after {timeout}s")
        finally:
            self.pending.pop(task_id, None)

    async def send_task_with_retry(
        self, topic: str, payload: dict, max_retries: int = 3, timeout: int = 30
    ) -> dict:
        for attempt in range(max_retries):
            try:
                return await self.send_task(topic, payload, timeout)
            except TimeoutError:
                if attempt < max_retries - 1:
                    self.logger.warning(
                        f"Retry {attempt + 1}/{max_retries} for topic {topic}"
                    )
                    payload["task_id"] = str(uuid.uuid4())
                    await asyncio.sleep(1)
                else:
                    raise
