import logging
import random
from typing import List, Optional

import nats


class AgentBid:
    def __init__(self, agent_id: str, cost: float, availability: float):
        self.agent_id = agent_id
        self.cost = cost
        self.availability = availability
        # Lower score wins: cheap + high-availability agent is preferred
        self.score = cost * (1 - availability)


class AuctionManager:
    def __init__(self, nc: nats.NATS):
        self.nc = nc
        self.logger = logging.getLogger("auction")

    async def run_auction(
        self, task_type: str, agent_pool: List[str], timeout: float = 2.0
    ) -> Optional[str]:
        """Simulate an auction where each agent submits a bid; lowest score wins."""
        if not agent_pool:
            self.logger.warning("Auction called with empty agent pool")
            return None

        bids: List[AgentBid] = []
        for agent_id in agent_pool:
            cost = random.uniform(0.1, 1.0)
            availability = random.uniform(0.5, 1.0)
            bid = AgentBid(agent_id=agent_id, cost=cost, availability=availability)
            bids.append(bid)
            self.logger.info(
                f"Bid from {agent_id}: cost={cost:.2f}, "
                f"availability={availability:.2f}, score={bid.score:.2f}"
            )

        winner = min(bids, key=lambda b: b.score)
        self.logger.info(
            f"Auction winner for task_type={task_type}: "
            f"{winner.agent_id} with score {winner.score:.2f}"
        )
        return winner.agent_id

    async def demo_auction(self) -> dict:
        """Run a demo auction with three simulated quote agents."""
        agents = ["quotes-agent-1", "quotes-agent-2", "quotes-agent-3"]
        winner = await self.run_auction("quotes", agents)
        return {
            "task_type": "quotes",
            "participants": agents,
            "winner": winner,
            "method": "lowest_cost_score",
        }
