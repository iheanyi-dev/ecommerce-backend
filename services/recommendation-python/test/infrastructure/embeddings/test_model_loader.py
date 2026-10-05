"""Tests for GTE model loading configuration."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import torch

from infrastructure.embeddings.model_loader import GTEModelLoader
from infrastructure.embeddings.model_manifest import GTE_MULTILINGUAL_BASE


def test_loader_uses_pinned_revision_and_cpu_precision(
    tmp_path: Path,
) -> None:
    """CPU loading uses the exact pinned revision without FP16."""
    tokenizer = MagicMock()
    model = MagicMock()

    with (
        patch(
            "infrastructure.embeddings.model_loader.AutoTokenizer"
            ".from_pretrained",
            return_value=tokenizer,
        ) as tokenizer_loader,
        patch(
            "infrastructure.embeddings.model_loader.AutoModel"
            ".from_pretrained",
            return_value=model,
        ) as model_loader,
        patch(
            "infrastructure.embeddings.model_loader.GTEModelLoader"
            "._resolve_device",
            return_value=torch.device("cpu"),
        ),
    ):
        loaded = GTEModelLoader(
            GTE_MULTILINGUAL_BASE,
            tmp_path / "models",
        ).load()

    cache_dir = str(tmp_path / "models")

    tokenizer_loader.assert_called_once_with(
        GTE_MULTILINGUAL_BASE.model_id,
        revision=GTE_MULTILINGUAL_BASE.revision,
        cache_dir=cache_dir,
        trust_remote_code=True,
    )

    model_loader.assert_called_once_with(
        GTE_MULTILINGUAL_BASE.model_id,
        revision=GTE_MULTILINGUAL_BASE.revision,
        cache_dir=cache_dir,
        trust_remote_code=True,
    )

    model.to.assert_called_once_with(torch.device("cpu"))
    model.eval.assert_called_once()

    assert loaded.tokenizer is tokenizer
    assert loaded.model is model
    assert loaded.device == torch.device("cpu")


def test_loader_uses_fp16_on_cuda(
    tmp_path: Path,
) -> None:
    """CUDA loading requests FP16 inference weights."""
    tokenizer = MagicMock()
    model = MagicMock()

    with (
        patch(
            "infrastructure.embeddings.model_loader.AutoTokenizer"
            ".from_pretrained",
            return_value=tokenizer,
        ),
        patch(
            "infrastructure.embeddings.model_loader.AutoModel"
            ".from_pretrained",
            return_value=model,
        ) as model_loader,
        patch(
            "infrastructure.embeddings.model_loader.GTEModelLoader"
            "._resolve_device",
            return_value=torch.device("cuda"),
        ),
    ):
        GTEModelLoader(
            GTE_MULTILINGUAL_BASE,
            tmp_path / "models",
        ).load()

    model_loader.assert_called_once_with(
        GTE_MULTILINGUAL_BASE.model_id,
        revision=GTE_MULTILINGUAL_BASE.revision,
        cache_dir=str(tmp_path / "models"),
        trust_remote_code=True,
        torch_dtype=torch.float16,
    )
