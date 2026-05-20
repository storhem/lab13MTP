import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from unittest.mock import MagicMock

import pytest


@pytest.mark.asyncio
async def test_auction_returns_winner():
    from auction import AuctionManager

    mock_nc = MagicMock()
    auction = AuctionManager(mock_nc)

    agents = ["agent-1", "agent-2", "agent-3"]
    winner = await auction.run_auction("quotes", agents)

    assert winner in agents


@pytest.mark.asyncio
async def test_auction_empty_pool():
    from auction import AuctionManager

    mock_nc = MagicMock()
    auction = AuctionManager(mock_nc)

    winner = await auction.run_auction("quotes", [])
    assert winner is None


@pytest.mark.asyncio
async def test_auction_single_agent():
    from auction import AuctionManager

    mock_nc = MagicMock()
    auction = AuctionManager(mock_nc)

    winner = await auction.run_auction("quotes", ["only-agent"])
    assert winner == "only-agent"
