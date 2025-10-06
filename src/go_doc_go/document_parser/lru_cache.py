import functools
import hashlib
import json
import logging
import os
import subprocess
from pathlib import Path
from threading import RLock

import time

logger = logging.getLogger(__name__)


class LRUCache:
    """Thread-safe LRU cache with time-based expiration."""

    def __init__(self, max_size=128, ttl=3600):
        """
        Initialize LRU cache.

        Args:
            max_size: Maximum number of items in cache
            ttl: Time to live in seconds (default: 1 hour)
        """
        self.cache = {}
        self.max_size = max_size
        self.ttl = ttl
        self.usage_order = []
        self._lock = RLock()

    def get(self, key):
        """Get item from cache if it exists and is not expired."""
        with self._lock:
            if key not in self.cache:
                return None

            value, timestamp = self.cache[key]
            current_time = time.time()

            # Check if item is expired
            if current_time - timestamp > self.ttl:
                # Remove expired item
                del self.cache[key]
                self.usage_order.remove(key)
                return None

            # Update usage order
            self.usage_order.remove(key)
            self.usage_order.append(key)

            return value

    def set(self, key, value):
        """Add item to cache with current timestamp."""
        with self._lock:
            # If key already exists, update usage order
            if key in self.cache:
                self.usage_order.remove(key)

            # If cache is full, evict least recently used item
            if len(self.cache) >= self.max_size and len(self.usage_order) > 0:
                lru_key = self.usage_order.pop(0)
                del self.cache[lru_key]

            # Add new item
            self.cache[key] = (value, time.time())
            self.usage_order.append(key)

    def clear(self):
        """Clear the cache."""
        with self._lock:
            self.cache.clear()
            self.usage_order.clear()


class GoLRUCache:
    """Go-based LRU cache implementation via subprocess calls."""

    def __init__(self, max_size=128, ttl=3600):
        """
        Initialize Go LRU cache wrapper.

        Args:
            max_size: Maximum number of items in cache
            ttl: Time to live in seconds (default: 1 hour)
        """
        self.max_size = max_size
        self.ttl = ttl
        self._lock = RLock()

        # Find the Go binary
        project_root = Path(__file__).parent.parent.parent.parent
        self.binary_path = project_root / "bin" / "lrucache"

        if not self.binary_path.exists():
            # Try alternative location
            self.binary_path = project_root / "go" / "bin" / "lrucache"

        if not self.binary_path.exists():
            raise FileNotFoundError(
                f"Go LRU cache binary not found. Expected at {self.binary_path}. "
                "Please build it with: cd go && go build -o ../bin/lrucache ./cmd/lrucache"
            )

        # Make cache file path unique to this instance
        cache_dir = Path.home() / ".go-doc-go" / "cache"
        # Use memory-mapped file for better performance
        self.cache_file = cache_dir / f"lru_mmap_{max_size}_{ttl}.bin"

    def _run_command(self, *args):
        """Run the Go binary with the given arguments."""
        cmd = [
            str(self.binary_path),
            *args,
            f"--max-size={self.max_size}",
            f"--ttl={self.ttl}",
            f"--cache-file={self.cache_file}",
            "--use-mmap=true"  # Enable memory-mapped file for better performance
        ]

        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=5  # 5 second timeout
            )

            if result.returncode != 0:
                logger.error(f"Go cache command failed: {result.stderr}")
                return None

            return result.stdout.strip()
        except subprocess.TimeoutExpired:
            logger.error("Go cache command timed out")
            return None
        except Exception as e:
            logger.error(f"Error running Go cache: {e}")
            return None

    def get(self, key):
        """Get item from cache if it exists and is not expired."""
        with self._lock:
            result = self._run_command("get", str(key))

            if result == "null" or result is None:
                return None

            try:
                return json.loads(result)
            except json.JSONDecodeError:
                # If not JSON, return as string
                return result

    def set(self, key, value):
        """Add item to cache with current timestamp."""
        with self._lock:
            # Convert value to JSON string
            if isinstance(value, str):
                # For empty strings, use JSON representation to avoid CLI parsing issues
                if value == "":
                    json_value = '""'
                else:
                    json_value = value
            else:
                try:
                    json_value = json.dumps(value)
                except (TypeError, ValueError):
                    json_value = str(value)

            result = self._run_command("set", str(key), json_value)
            return result == "OK"

    def clear(self):
        """Clear the cache."""
        with self._lock:
            result = self._run_command("clear")
            return result == "OK"


def create_lru_cache(max_size=128, ttl=3600):
    """
    Factory function to create LRU cache instance.

    Uses Go implementation if USE_GO_MODULES or USE_GO_CACHE environment variable is set,
    otherwise uses Python implementation.

    Args:
        max_size: Maximum number of items in cache
        ttl: Time to live in seconds

    Returns:
        LRUCache or GoLRUCache instance
    """
    # Check if Go modules should be used (unified flag or specific flag)
    use_go_modules = os.environ.get("USE_GO_MODULES", "").lower() in ("true", "1", "yes")
    use_go_cache = os.environ.get("USE_GO_CACHE", "").lower() in ("true", "1", "yes")
    use_go = use_go_modules or use_go_cache

    if use_go:
        try:
            cache = GoLRUCache(max_size=max_size, ttl=ttl)
            logger.info("Using Go LRU cache implementation")
            return cache
        except FileNotFoundError as e:
            logger.warning(f"Go cache not available: {e}. Falling back to Python implementation.")

    logger.debug("Using Python LRU cache implementation")
    return LRUCache(max_size=max_size, ttl=ttl)


def ttl_cache(maxsize=128, ttl=3600):
    """
    Decorator that caches function results with a time-to-live (TTL).

    Args:
        maxsize: Maximum cache size
        ttl: Time to live in seconds
    """
    cache = create_lru_cache(max_size=maxsize, ttl=ttl)
    lock = RLock()

    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            # Create a cache key from the function name and arguments
            key_parts = [func.__name__]
            key_parts.extend([str(arg) for arg in args])
            key_parts.extend([f"{k}={v}" for k, v in sorted(kwargs.items())])
            key = hashlib.md5(":".join(key_parts).encode()).hexdigest()

            # Try to get from cache
            with lock:
                result = cache.get(key)
                if result is not None:
                    logger.debug(f"Cache hit for {func.__name__}")
                    return result

                # Call the function
                result = func(*args, **kwargs)

                # Store in cache
                cache.set(key, result)
                return result

        # Add clear_cache method
        def clear_cache():
            with lock:
                cache.clear()

        wrapper.clear_cache = clear_cache
        return wrapper

    return decorator
