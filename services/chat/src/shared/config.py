"""Validated configuration loaded from environment and the service .env file."""

from functools import lru_cache
from pathlib import Path

from pydantic import Field, SecretStr
from pydantic_settings import BaseSettings, SettingsConfigDict


SERVICE_ROOT = Path(__file__).resolve().parents[2]


class Settings(BaseSettings):
    """Runtime settings for the internal Chat service."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=True,
        extra="ignore",
    )

    CHAT_SERVICE_HOST: str = "0.0.0.0"
    CHAT_SERVICE_PORT: int = Field(default=8004, ge=1, le=65535)
    CHAT_LOG_LEVEL: str = "INFO"

    CHAT_DATABASE_URL: str = (
        "postgresql+asyncpg://chat:chat@localhost:5432/chat"
    )

    GEMINI_API_KEY: SecretStr = SecretStr("")
    GEMINI_MODEL: str = "gemini-2.5-flash"

    CHAT_SYSTEM_PROMPT_FILE: str = "prompts/system.md"

    CHAT_AGENT_MAX_TOOL_ROUNDS: int = Field(default=8, ge=1)
    CHAT_CONVERSATION_HISTORY_LIMIT: int = Field(default=20, ge=1)

    # Internal service authentication. Configure these through .env.
    CHAT_SERVICE_NAME: str = "chat"
    CHAT_SERVICE_SECRET: SecretStr = SecretStr("")
    CHAT_CALLER_SERVICE_NAME: str = "store"
    CHAT_CALLER_SERVICE_SECRET: SecretStr = SecretStr("")

    # Downstream addresses are configuration, not source-code constants.
    EMBEDDING_SERVICE_ADDR: str = ""
    EMBEDDING_SERVICE_NAME: str = "embedding"
    EMBEDDING_SERVICE_SECRET: SecretStr = SecretStr("")

    PRODUCT_SERVICE_ADDR: str = ""
    PRODUCT_SERVICE_NAME: str = "product"
    PRODUCT_SERVICE_SECRET: SecretStr = SecretStr("")

    CHAT_GRPC_MAX_WORKERS: int = Field(default=10, ge=1)

    def load_system_prompt(self) -> str:
        """Load the service-owned system prompt from its configured file."""
        configured_path = Path(self.CHAT_SYSTEM_PROMPT_FILE)
        prompt_path = (
            configured_path
            if configured_path.is_absolute()
            else SERVICE_ROOT / configured_path
        )

        try:
            prompt = prompt_path.read_text(encoding="utf-8").strip()
        except OSError as exc:
            raise RuntimeError(
                f"Unable to load Chat system prompt: {prompt_path}"
            ) from exc

        if not prompt:
            raise RuntimeError("Chat system prompt must not be empty")

        return prompt


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    """Return cached validated runtime settings."""
    return Settings()
