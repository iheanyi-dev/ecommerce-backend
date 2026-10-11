"""Unit tests for lazy GTE model loading and embedding generation."""

from types import SimpleNamespace

import pytest
import torch

from domain.value_objects import EmbeddingModelVersion
from infrastructure.embeddings import (
    EmbeddingModelManifest,
    GTEEmbeddingModel,
)
from infrastructure.embeddings import gte_embedding_model as adapter_module


class FakeTokenizer:
    """Return token tensors without invoking a real tokenizer."""

    def __call__(self, text, **kwargs):
        assert text == "sample knowledge text"
        assert kwargs["max_length"] == 8192
        return {"input_ids": torch.tensor([[1, 2, 3]])}


class FakeModel:
    """Expose a predictable CLS vector for deterministic unit tests."""

    def __init__(self):
        self.device = None
        self.evaluation_mode = False

    def to(self, device):
        self.device = device
        return self

    def eval(self):
        self.evaluation_mode = True
        return self

    def __call__(self, **kwargs):
        return SimpleNamespace(
            # The first token is [3, 4, 0], whose normalized vector is
            # [0.6, 0.8, 0]. Other token vectors must not affect CLS pooling.
            last_hidden_state=torch.tensor(
                [[[3.0, 4.0, 0.0], [100.0, 100.0, 100.0]]]
            )
        )


@pytest.fixture
def setup_adapter(monkeypatch):
    manifest = EmbeddingModelManifest(
        model_id="test/model",
        revision="test-revision",
        embedding_dimension=3,
    )
    tokenizer = FakeTokenizer()
    model = FakeModel()
    calls = {"tokenizer": 0, "model": 0}

    def load_tokenizer(*args, **kwargs):
        calls["tokenizer"] += 1
        assert args == ("test/model",)
        assert kwargs["revision"] == "test-revision"
        return tokenizer

    def load_model(*args, **kwargs):
        calls["model"] += 1
        assert args == ("test/model",)
        assert kwargs["revision"] == "test-revision"
        return model

    monkeypatch.setattr(
        adapter_module.AutoTokenizer,
        "from_pretrained",
        load_tokenizer,
    )
    monkeypatch.setattr(
        adapter_module.AutoModel,
        "from_pretrained",
        load_model,
    )
    monkeypatch.setattr(
        adapter_module.torch.cuda,
        "is_available",
        lambda: False,
    )

    return GTEEmbeddingModel(manifest), model, calls


def test_model_is_not_loaded_during_construction(setup_adapter):
    adapter, _, calls = setup_adapter

    assert calls == {"tokenizer": 0, "model": 0}
    assert adapter._loaded is None


def test_first_request_loads_model_and_generates_normalized_cls_embedding(
    setup_adapter,
):
    adapter, model, calls = setup_adapter

    embedding, version = adapter.generate("sample knowledge text")

    assert embedding.vector == pytest.approx((0.6, 0.8, 0.0))
    assert sum(value * value for value in embedding.vector) == pytest.approx(1.0)
    assert embedding.dimension == 3
    assert version == EmbeddingModelVersion("test/model@test-revision")
    assert calls == {"tokenizer": 1, "model": 1}
    assert model.evaluation_mode is True


def test_subsequent_requests_reuse_the_loaded_model(setup_adapter):
    adapter, _, calls = setup_adapter

    adapter.generate("sample knowledge text")
    adapter.generate("sample knowledge text")

    assert calls == {"tokenizer": 1, "model": 1}


@pytest.mark.parametrize("text", ["", "   ", "\n\t"])
def test_rejects_empty_text_without_loading_model(setup_adapter, text):
    adapter, _, calls = setup_adapter

    with pytest.raises(ValueError, match="must not be empty"):
        adapter.generate(text)

    assert calls == {"tokenizer": 0, "model": 0}


def test_rejects_non_string_input(setup_adapter):
    adapter, _, calls = setup_adapter

    with pytest.raises(TypeError, match="must be a string"):
        adapter.generate(None)

    assert calls == {"tokenizer": 0, "model": 0}


def test_rejects_embedding_with_wrong_dimension(setup_adapter, monkeypatch):
    adapter, _, _ = setup_adapter

    # Simulate a model returning a vector incompatible with its manifest.
    monkeypatch.setattr(
        adapter_module.functional,
        "normalize",
        lambda vector, p, dim: torch.tensor([[1.0, 0.0]]),
    )

    with pytest.raises(ValueError, match="dimension does not match"):
        adapter.generate("sample knowledge text")
