"""Tests for Recommendation HTTP routes."""

from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID, uuid4

from fastapi.testclient import TestClient

from src.presentation.http.app import create_app


SERVICE_NAME = "gateway"
SERVICE_SECRET = "test-secret"


@dataclass
class StoreRecommendation:
    store_id: UUID
    score: float


@dataclass
class ProductRecommendation:
    product_id: UUID
    score: float


class StubRecommendationResult:
    def __init__(self, recommendations):
        self.recommendations = tuple(recommendations)


class StubStores:
    def __init__(self):
        self.inputs = []

    def execute(self, input):
        self.inputs.append(input)
        return StubRecommendationResult(
            [StoreRecommendation(uuid4(), 0.95)]
        )


class StubProducts:
    def __init__(self):
        self.inputs = []

    def execute(self, input):
        self.inputs.append(input)
        return StubRecommendationResult(
            [ProductRecommendation(uuid4(), 0.91)]
        )


class StubLogger:
    def info(self, event, **fields):
        pass

    def warning(self, event, **fields):
        pass

    def error(self, event, **fields):
        pass


class StubMetrics:
    def increment(self, name, value=1):
        pass

    def observe(self, name, value):
        pass


def create_test_client():
    stores = StubStores()
    products = StubProducts()

    app = create_app(
        service_name=SERVICE_NAME,
        service_secret=SERVICE_SECRET,
        recommend_stores=stores,
        recommend_products=products,
        logger=StubLogger(),
        metrics=StubMetrics(),
    )

    return TestClient(app), stores, products


def test_store_recommendation_requires_service_auth():
    client, _, _ = create_test_client()

    response = client.get(
        "/api/v1/recommendations/stores",
        params={"query": "shoes"},
    )

    assert response.status_code == 401


def test_store_recommendation_returns_results():
    client, stores, _ = create_test_client()

    response = client.get(
        "/api/v1/recommendations/stores",
        params={"query": "shoes"},
        headers={
            "X-Service-Name": SERVICE_NAME,
            "X-Service-Secret": SERVICE_SECRET,
        },
    )

    assert response.status_code == 200
    assert len(response.json()["recommendations"]) == 1
    assert stores.inputs[0].query == "shoes"


def test_product_recommendation_returns_results():
    client, _, products = create_test_client()

    product_id = uuid4()

    response = client.get(
        "/api/v1/recommendations/products",
        params={"product_id": str(product_id)},
        headers={
            "X-Service-Name": SERVICE_NAME,
            "X-Service-Secret": SERVICE_SECRET,
        },
    )

    assert response.status_code == 200
    assert len(response.json()["recommendations"]) == 1
    assert products.inputs[0].product_id == product_id


def test_recommendation_requires_an_input():
    client, _, _ = create_test_client()

    response = client.get(
        "/api/v1/recommendations/stores",
        headers={
            "X-Service-Name": SERVICE_NAME,
            "X-Service-Secret": SERVICE_SECRET,
        },
    )

    assert response.status_code == 400


def test_request_id_is_returned():
    client, _, _ = create_test_client()

    request_id = str(uuid4())

    response = client.get(
        "/api/v1/recommendations/stores",
        params={"query": "shoes"},
        headers={
            "X-Service-Name": SERVICE_NAME,
            "X-Service-Secret": SERVICE_SECRET,
            "X-Request-ID": request_id,
        },
    )

    assert response.status_code == 200
