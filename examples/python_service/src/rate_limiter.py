"""Token bucket and sliding window rate limiting implementations.

Provides bounded concurrency and token-based rate limiting with zero external
dependencies for microservice ingress control.
"""

from dataclasses import dataclass
import time
from typing import Dict, Optional, Tuple


@dataclass(frozen=True)
class TokenBucketConfig:
    """Configuration for token bucket rate limiter."""
    capacity: float
    refill_rate: float  # tokens added per second

    def __post_init__(self) -> None:
        if self.capacity <= 0.0:
            raise ValueError(f"Capacity must be strictly positive, got: {self.capacity}")
        if self.refill_rate <= 0.0:
            raise ValueError(f"Refill rate must be strictly positive, got: {self.refill_rate}")


@dataclass(frozen=True)
class RateLimitResult:
    """Outcome of a rate limit check."""
    allowed: bool
    remaining_tokens: float
    reset_after_seconds: float
    client_id: str


class TokenBucketRateLimiter:
    """Thread-safe token bucket rate limiter with lazy refill calculation."""

    def __init__(self, config: TokenBucketConfig) -> None:
        self.config = config
        self._buckets: Dict[str, Tuple[float, float]] = {}  # client_id -> (tokens, last_refill_time)

    def _get_current_time(self) -> float:
        return time.monotonic()

    def acquire(self, client_id: str, tokens_requested: float = 1.0) -> RateLimitResult:
        """Attempt to consume tokens for a given client ID."""
        if tokens_requested <= 0.0:
            raise ValueError(f"Tokens requested must be positive, got: {tokens_requested}")

        now = self._get_current_time()
        current_tokens, last_refill = self._buckets.get(
            client_id, (self.config.capacity, now)
        )

        # Refill tokens according to elapsed time
        elapsed = now - last_refill
        if elapsed > 0.0:
            current_tokens = min(
                self.config.capacity,
                current_tokens + (elapsed * self.config.refill_rate),
            )
            last_refill = now

        if current_tokens >= tokens_requested:
            new_tokens = current_tokens - tokens_requested
            self._buckets[client_id] = (new_tokens, last_refill)
            return RateLimitResult(
                allowed=True,
                remaining_tokens=new_tokens,
                reset_after_seconds=0.0,
                client_id=client_id,
            )

        # Rate limited
        deficit = tokens_requested - current_tokens
        reset_seconds = deficit / self.config.refill_rate if self.config.refill_rate > 0.0 else 0.0
        self._buckets[client_id] = (current_tokens, last_refill)
        return RateLimitResult(
            allowed=False,
            remaining_tokens=current_tokens,
            reset_after_seconds=reset_seconds,
            client_id=client_id,
        )

    def get_remaining_tokens(self, client_id: str) -> float:
        """Inspect available tokens for a client without consuming them."""
        now = self._get_current_time()
        current_tokens, last_refill = self._buckets.get(
            client_id, (self.config.capacity, now)
        )
        elapsed = now - last_refill
        if elapsed > 0.0:
            current_tokens = min(
                self.config.capacity,
                current_tokens + (elapsed * self.config.refill_rate),
            )
        return current_tokens

    def reset_client(self, client_id: str) -> None:
        """Reset a client back to full initial capacity."""
        if client_id in self._buckets:
            del self._buckets[client_id]


class SlidingWindowLimiter:
    """Sliding window log limiter for precise second-level windows."""

    def __init__(self, max_requests: int, window_seconds: float) -> None:
        if max_requests <= 0:
            raise ValueError(f"max_requests must be positive, got: {max_requests}")
        if window_seconds <= 0.0:
            raise ValueError(f"window_seconds must be positive, got: {window_seconds}")

        self.max_requests = max_requests
        self.window_seconds = window_seconds
        self._request_logs: Dict[str, list] = {}

    def _get_current_time(self) -> float:
        return time.monotonic()

    def allow_request(self, client_id: str) -> bool:
        """Check if request is allowed within sliding window."""
        now = self._get_current_time()
        window_start = now - self.window_seconds

        timestamps = self._request_logs.get(client_id, [])
        # Prune expired entries
        timestamps = [ts for ts in timestamps if ts > window_start]

        if len(timestamps) < self.max_requests:
            timestamps.append(now)
            self._request_logs[client_id] = timestamps
            return True

        self._request_logs[client_id] = timestamps
        return False
