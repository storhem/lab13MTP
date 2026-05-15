import asyncio
import logging
import signal

from agent import LLMAnalystAgent

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(name)s %(levelname)s %(message)s",
)
logger = logging.getLogger("llm-analyst.main")


async def main():
    agent = LLMAnalystAgent()
    task = asyncio.create_task(agent.run())

    loop = asyncio.get_running_loop()

    def _shutdown():
        logger.info("Received shutdown signal")
        task.cancel()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, _shutdown)

    try:
        await task
    except asyncio.CancelledError:
        logger.info("llm-analyst main task cancelled")


if __name__ == "__main__":
    asyncio.run(main())
