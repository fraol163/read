"""Comprehensive unit tests for TokenBucketRateLimiter and SlidingWindowLimiter."""

import time
import pytest
from src.rate_limiter import (
    TokenBucketConfig,
    RateLimitResult,
    TokenBucketRateLimiter,
    SlidingWindowLimiter,
)


def test_config_validation():
    """Verify invalid configurations are rejected with ValueError."""
    with pytest.raises(ValueError):
        TokenBucketConfig(capacity=0.0, refill_rate=1.0)

    with pytest.raises(ValueError):
        TokenBucketConfig(capacity=10.0, refill_rate=-1.0)


def test_token_bucket_basic_acquire():
    """Verify single and multiple token acquisitions."""
    cfg = TokenBucketConfig(capacity=10.0, refill_rate=2.0)
    limiter = TokenBucketRateLimiter(cfg)

    # First acquire
    res1 = limiter.acquire("user-1", tokens_requested=3.0)
    assert res1.allowed is True
    assert res1.remaining_tokens == pytest.approx(7.0, rel=1e-3)
    assert res1.client_id == "user-1"
    assert res1.reset_after_seconds == 0.0

    # Second acquire
    res2 = limiter.acquire("user-1", tokens_requested=5.0)
    assert res2.allowed is True
    assert res2.remaining_tokens == pytest.approx(2.0, rel=1e-3)

    # Third acquire exceeds remaining capacity
    res3 = limiter.acquire("user-1", tokens_requested=3.0)
    assert res3.allowed is False
    assert res3.remaining_tokens == pytest.approx(2.0, rel=1e-3)
    assert res3.reset_after_seconds > 0.0


def test_token_bucket_isolation_between_clients():
    """Verify separate clients maintain independent token balances."""
    cfg = TokenBucketConfig(capacity=5.0, refill_rate=1.0)
    limiter = TokenBucketRateLimiter(cfg)

    res_a = limiter.acquire("client-alpha", tokens_requested=5.0)
    assert res_a.allowed is True

    # Client alpha should now be exhausted
    res_a2 = limiter.acquire("client-alpha", tokens_requested=1.0)
    assert res_a2.allowed is False

    # Client beta should still have full capacity
    res_b = limiter.acquire("client-beta", tokens_requested=5.0)
    assert res_b.allowed is True


def test_token_bucket_invalid_request():
    """Verify requesting zero or negative tokens raises ValueError."""
    cfg = TokenBucketConfig(capacity=10.0, refill_rate=1.0)
    limiter = TokenBucketRateLimiter(cfg)

    with pytest.raises(ValueError):
        limiter.acquire("client-1", tokens_requested=0.0)

    with pytest.raises(ValueError):
        limiter.acquire("client-1", tokens_requested=-2.0)


def test_token_bucket_refill_simulation(monkeypatch):
    """Verify tokens refill properly as simulated time progresses."""
    current_mock_time = 1000.0

    cfg = TokenBucketConfig(capacity=10.0, refill_rate=5.0)
    limiter = TokenBucketRateLimiter(cfg)
    monkeypatch.setattr(limiter, "_get_current_time", lambda: current_mock_time)

    # Consume all 10 tokens
    res = limiter.acquire("user-refill", tokens_requested=10.0)
    assert res.allowed is True
    assert res.remaining_tokens == 0.0

    # Advance time by 1.0 second -> should refill 5 tokens
    current_mock_time += 1.0
    assert limiter.get_remaining_tokens("user-refill") == 5.0

    # Advance time by 2.0 more seconds -> should cap at capacity 10.0
    current_mock_time += 2.0
    assert limiter.get_remaining_tokens("user-refill") == 10.0


def test_token_bucket_reset():
    """Verify reset_client restores full capacity."""
    cfg = TokenBucketConfig(capacity=5.0, refill_rate=1.0)
    limiter = TokenBucketRateLimiter(cfg)

    limiter.acquire("user-reset", tokens_requested=5.0)
    assert limiter.get_remaining_tokens("user-reset") == pytest.approx(0.0, abs=1e-3)

    limiter.reset_client("user-reset")
    assert limiter.get_remaining_tokens("user-reset") == pytest.approx(5.0, rel=1e-3)


def test_sliding_window_validation():
    """Verify invalid sliding window params raise ValueError."""
    with pytest.raises(ValueError):
        SlidingWindowLimiter(max_requests=0, window_seconds=1.0)

    with pytest.raises(ValueError):
        SlidingWindowLimiter(max_requests=5, window_seconds=0.0)


def test_sliding_window_allow_and_reject(monkeypatch):
    """Verify sliding window bounds requests within timeframe."""
    current_mock_time = 500.0
    window = SlidingWindowLimiter(max_requests=2, window_seconds=10.0)
    monkeypatch.setattr(window, "_get_current_time", lambda: current_mock_time)

    assert window.allow_request("c1") is True
    assert window.allow_request("c1") is True
    assert window.allow_request("c1") is False

    # Advance time past window (11 seconds)
    current_mock_time += 11.0
    assert window.allow_request("c1") is True
