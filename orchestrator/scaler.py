import asyncio
import logging
import random

try:
    import docker
    _docker_available = True
except ImportError:
    _docker_available = False


class DynamicScaler:
    def __init__(self, check_interval: int = 10, queue_threshold: int = 5):
        self.check_interval = check_interval
        self.queue_threshold = queue_threshold
        self.logger = logging.getLogger("scaler")
        self._running = False

        if _docker_available:
            try:
                self.docker_client = docker.from_env()
            except Exception:
                self.docker_client = None
                self.logger.warning("Docker client unavailable, scaler in mock mode")
        else:
            self.docker_client = None
            self.logger.warning("docker-py not installed, scaler in mock mode")

    async def get_queue_length(self) -> int:
        """Return a simulated queue depth (0–10).

        In a real JetStream deployment this would query the consumer info via NATS management API.
        """
        return random.randint(0, 10)

    async def scale_up(self, service: str):
        if self.docker_client:
            try:
                containers = self.docker_client.containers.list(filters={"name": service})
                current = len(containers)
                self.logger.info(
                    f"Scaling up {service}: currently {current} instance(s)"
                )
                # In a real Swarm/Compose scenario:
                # svc = self.docker_client.services.get(service)
                # svc.scale(current + 1)
            except Exception as e:
                self.logger.error(f"Scale up failed for {service}: {e}")
        else:
            self.logger.info(f"[MOCK] Would scale up service '{service}'")

    async def run(self):
        self._running = True
        self.logger.info(
            f"DynamicScaler started (interval={self.check_interval}s, "
            f"threshold={self.queue_threshold})"
        )
        while self._running:
            queue_len = await self.get_queue_length()
            self.logger.debug(f"Queue length: {queue_len}")
            if queue_len > self.queue_threshold:
                self.logger.info(
                    f"Queue length {queue_len} > threshold {self.queue_threshold}, scaling up"
                )
                await self.scale_up("quotes-agent")
            await asyncio.sleep(self.check_interval)

    def stop(self):
        self._running = False
        self.logger.info("DynamicScaler stopping")
