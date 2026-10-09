"""HTTP service-to-service authentication."""

from __future__ import annotations

import secrets

from fastapi import Header, HTTPException, status


def require_service_auth(
    expected_service_name: str,
    expected_service_secret: str,
):
    """Create a FastAPI dependency enforcing internal service authentication."""

    def dependency(
        service_name: str | None = Header(default=None, alias="X-Service-Name"),
        service_secret: str | None = Header(
            default=None,
            alias="X-Service-Secret",
        ),
    ) -> None:
        if service_name != expected_service_name:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="unauthorized",
            )

        if service_secret is None or not secrets.compare_digest(
            service_secret,
            expected_service_secret,
        ):
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="unauthorized",
            )

    return dependency
