"""Tests for the external Product recommendation use case."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import uuid4

from application.dto.find_similar_products import ProductRecommendation
from application.dto.recommend_products import RecommendProductsInput
from application.use_cases.recommend_products import RecommendProducts


@dataclass
class StubFindSimilarProducts:
    """Test double for the Product similarity candidate use case."""

    recommendations: tuple[ProductRecommendation, ...]
    calls: int = 0

    def execute(self, input):
        self.calls += 1
        self.input = input

        from application.dto.find_similar_products import (
            FindSimilarProductsOutput,
        )

        return FindSimilarProductsOutput(
            recommendations=self.recommendations,
        )


@dataclass
class StubFindAssociatedProducts:
    """Test double for the Product behavioral candidate use case."""

    recommendations: tuple[ProductRecommendation, ...]
    calls: int = 0

    def execute(self, input):
        self.calls += 1
        self.input = input

        from application.dto.find_associated_products import (
            FindAssociatedProductsOutput,
        )

        return FindAssociatedProductsOutput(
            recommendations=self.recommendations,
        )


@dataclass
class StubReranker:
    """Test double for the Product reranking port."""

    result: list[ProductRecommendation]
    calls: int = 0

    def rerank_stores(
        self,
        similar_stores,
        associated_stores,
    ):
        raise AssertionError(
            "Store reranking should not be called by RecommendProducts"
        )

    def rerank_products(
        self,
        similar_products,
        associated_products,
    ):
        self.calls += 1
        self.similar_products = similar_products
        self.associated_products = associated_products

        return self.result


def test_query_calls_only_similarity_candidate_use_case():
    product_id = uuid4()

    similar = (
        ProductRecommendation(
            product_id=product_id,
            score=0.91,
        ),
    )

    find_similar = StubFindSimilarProducts(
        recommendations=similar,
    )
    find_associated = StubFindAssociatedProducts(
        recommendations=(),
    )
    reranker = StubReranker(
        result=list(similar),
    )

    use_case = RecommendProducts(
        find_similar_products=find_similar,
        find_associated_products=find_associated,
        reranker=reranker,
    )

    output = use_case.execute(
        RecommendProductsInput(
            query="wireless headphones",
        )
    )

    assert find_similar.calls == 1
    assert find_associated.calls == 0
    assert reranker.calls == 1
    assert reranker.similar_products == list(similar)
    assert reranker.associated_products == []
    assert output.recommendations == similar


def test_product_id_calls_only_behavioral_candidate_use_case():
    product_id = uuid4()

    associated = (
        ProductRecommendation(
            product_id=product_id,
            score=0.82,
        ),
    )

    find_similar = StubFindSimilarProducts(
        recommendations=(),
    )
    find_associated = StubFindAssociatedProducts(
        recommendations=associated,
    )
    reranker = StubReranker(
        result=list(associated),
    )

    use_case = RecommendProducts(
        find_similar_products=find_similar,
        find_associated_products=find_associated,
        reranker=reranker,
    )

    output = use_case.execute(
        RecommendProductsInput(
            product_id=product_id,
        )
    )

    assert find_similar.calls == 0
    assert find_associated.calls == 1
    assert reranker.calls == 1
    assert reranker.similar_products == []
    assert reranker.associated_products == list(associated)
    assert output.recommendations == associated


def test_query_and_product_id_call_both_candidate_use_cases():
    similar_product_id = uuid4()
    associated_product_id = uuid4()

    similar = (
        ProductRecommendation(
            product_id=similar_product_id,
            score=0.91,
        ),
    )

    associated = (
        ProductRecommendation(
            product_id=associated_product_id,
            score=0.83,
        ),
    )

    find_similar = StubFindSimilarProducts(
        recommendations=similar,
    )
    find_associated = StubFindAssociatedProducts(
        recommendations=associated,
    )
    reranker = StubReranker(
        result=[associated[0], similar[0]],
    )

    use_case = RecommendProducts(
        find_similar_products=find_similar,
        find_associated_products=find_associated,
        reranker=reranker,
    )

    output = use_case.execute(
        RecommendProductsInput(
            query="wireless headphones",
            product_id=similar_product_id,
        )
    )

    assert find_similar.calls == 1
    assert find_associated.calls == 1
    assert reranker.similar_products == list(similar)
    assert reranker.associated_products == list(associated)
    assert output.recommendations == (associated[0], similar[0])


def test_missing_query_returns_empty_similarity_candidates():
    find_similar = StubFindSimilarProducts(
        recommendations=(),
    )
    find_associated = StubFindAssociatedProducts(
        recommendations=(),
    )
    reranker = StubReranker(
        result=[],
    )

    use_case = RecommendProducts(
        find_similar_products=find_similar,
        find_associated_products=find_associated,
        reranker=reranker,
    )

    use_case.execute(
        RecommendProductsInput(
            product_id=uuid4(),
        )
    )

    assert reranker.similar_products == []


def test_missing_product_id_returns_empty_behavioral_candidates():
    find_similar = StubFindSimilarProducts(
        recommendations=(),
    )
    find_associated = StubFindAssociatedProducts(
        recommendations=(),
    )
    reranker = StubReranker(
        result=[],
    )

    use_case = RecommendProducts(
        find_similar_products=find_similar,
        find_associated_products=find_associated,
        reranker=reranker,
    )

    use_case.execute(
        RecommendProductsInput(
            query="wireless headphones",
        )
    )

    assert reranker.associated_products == []


def test_no_inputs_returns_empty_candidates_to_reranker():
    find_similar = StubFindSimilarProducts(
        recommendations=(),
    )
    find_associated = StubFindAssociatedProducts(
        recommendations=(),
    )
    reranker = StubReranker(
        result=[],
    )

    use_case = RecommendProducts(
        find_similar_products=find_similar,
        find_associated_products=find_associated,
        reranker=reranker,
    )

    output = use_case.execute(
        RecommendProductsInput()
    )

    assert find_similar.calls == 0
    assert find_associated.calls == 0
    assert reranker.calls == 1
    assert reranker.similar_products == []
    assert reranker.associated_products == []
    assert output.recommendations == ()
