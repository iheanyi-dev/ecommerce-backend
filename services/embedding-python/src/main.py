"""Executable entrypoint for the Embedding service."""

from __future__ import annotations

import asyncio
import logging

from infrastructure.runtime import build_runtime, run_runtime


def main() -> None:
    """Build the runtime, run workers, and always release owned resources."""
    logging.basicConfig(level=logging.INFO)
    runtime = build_runtime()

    try:
        asyncio.run(run_runtime(runtime))
    finally:
        runtime.close()


if __name__ == "__main__":
    main()
