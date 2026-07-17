"""
Kasumi Engine — Retrieval Service Configuration

Pydantic-based configuration with YAML file loading and environment variable
overrides. Mirrors the shared JSON Schema in config/schema.json.

Configuration precedence (highest to lowest):
    1. Environment variables (KASUMI_* prefix)
    2. Configuration file (YAML)
    3. Built-in defaults
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Literal

import yaml
from pydantic import BaseModel, Field, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class IngestionConfig(BaseModel):
    """Go-based gRPC ingestion service configuration."""

    host: str = "0.0.0.0"
    port: int = Field(default=50051, ge=1, le=65535)
    log_level: Literal["debug", "info", "warn", "error"] = "info"
    max_concurrent_streams: int = Field(default=1000, ge=1, le=10000)
    backpressure_buffer_size: int = Field(default=10000, ge=100, le=1000000)
    rate_limit_rps: int = Field(default=5000, ge=1)
    tls_enabled: bool = False
    tls_cert_path: str = ""
    tls_key_path: str = ""

    @model_validator(mode="after")
    def validate_tls(self) -> IngestionConfig:
        """TLS requires both cert and key paths."""
        if self.tls_enabled and (not self.tls_cert_path or not self.tls_key_path):
            raise ValueError(
                "tls_enabled requires both tls_cert_path and tls_key_path"
            )
        return self


class RetrievalConfig(BaseModel):
    """Python FastAPI retrieval service configuration."""

    host: str = "0.0.0.0"
    port: int = Field(default=8000, ge=1, le=65535)
    log_level: Literal["debug", "info", "warn", "error"] = "info"
    candidate_count: int = Field(default=500, ge=10, le=5000)
    latency_budget_ms: int = Field(default=20, ge=1, le=1000)
    index_type: Literal["faiss", "milvus"] = "faiss"
    faiss_nlist: int = Field(default=256, ge=1)
    faiss_nprobe: int = Field(default=16, ge=1)
    embedding_dim: int = Field(default=128, ge=1)


class RerankingConfig(BaseModel):
    """C++/TensorRT reranking service configuration."""

    host: str = "0.0.0.0"
    port: int = Field(default=8001, ge=1, le=65535)
    latency_budget_ms: int = Field(default=30, ge=1, le=1000)
    model_type: Literal["lightgbm", "two_tower"] = "lightgbm"
    triton_url: str = "localhost:8002"
    fallback_enabled: bool = True
    dynamic_batching_max_batch_size: int = Field(default=32, ge=1, le=512)


class RedisConfig(BaseModel):
    """Redis feature store configuration."""

    host: str = "localhost"
    port: int = Field(default=6379, ge=1, le=65535)
    password: str = ""
    db: int = Field(default=0, ge=0, le=15)
    max_connections: int = Field(default=50, ge=1, le=1000)
    key_prefix: str = "kasumi:"
    feature_ttl_seconds: int = Field(default=3600, ge=60)
    session_ttl_seconds: int = Field(default=1800, ge=60)


class PrivacyConfig(BaseModel):
    """APPI privacy configuration — differential privacy and anonymization."""

    epsilon: float = Field(default=1.0, gt=0, le=10.0)
    sensitivity: float = Field(default=1.0, gt=0)
    salt_rotation_hours: int = Field(default=24, ge=1, le=168)
    budget_ceiling: float = Field(default=10.0, gt=0)
    budget_alert_threshold: float = Field(default=0.8, gt=0, lt=1.0)
    aggregation_window_minutes: int = Field(default=15, ge=1, le=1440)
    redact_session_ids_in_logs: bool = True


class ObservabilityConfig(BaseModel):
    """Prometheus, Grafana, and tracing configuration."""

    metrics_enabled: bool = True
    metrics_port: int = Field(default=9090, ge=1, le=65535)
    tracing_enabled: bool = False
    log_format: Literal["json", "text"] = "json"


class KasumiConfig(BaseSettings):
    """
    Root configuration for the Kasumi Engine.

    Supports loading from YAML files and environment variable overrides
    with the KASUMI_ prefix. Environment variables use double-underscore
    to separate nested fields (e.g., KASUMI_REDIS__HOST).
    """

    model_config = SettingsConfigDict(
        env_prefix="KASUMI_",
        env_nested_delimiter="__",
        case_sensitive=False,
    )

    ingestion: IngestionConfig = IngestionConfig()
    retrieval: RetrievalConfig = RetrievalConfig()
    reranking: RerankingConfig = RerankingConfig()
    redis: RedisConfig = RedisConfig()
    privacy: PrivacyConfig = PrivacyConfig()
    observability: ObservabilityConfig = ObservabilityConfig()


def load_config(path: str | Path | None = None) -> KasumiConfig:
    """
    Load configuration from a YAML file with environment variable overrides.

    Args:
        path: Path to the YAML configuration file. If None, checks
              KASUMI_CONFIG_PATH env var, then falls back to defaults.

    Returns:
        Validated KasumiConfig instance.

    Raises:
        FileNotFoundError: If the specified config file does not exist.
        ValueError: If the configuration fails validation.
    """
    # Resolve config path
    if path is None:
        path = os.environ.get("KASUMI_CONFIG_PATH")

    # Load YAML if path is provided
    yaml_data: dict = {}
    if path is not None:
        config_path = Path(path)
        if not config_path.exists():
            raise FileNotFoundError(f"Configuration file not found: {config_path}")

        with open(config_path) as f:
            yaml_data = yaml.safe_load(f) or {}

    # Build config from YAML data — env vars are applied automatically
    # by pydantic-settings
    try:
        config = KasumiConfig(**yaml_data)
    except Exception as e:
        raise ValueError(f"Configuration validation failed: {e}") from e

    return config


def load_config_from_dict(data: dict) -> KasumiConfig:
    """
    Load configuration from a dictionary. Useful for testing.

    Args:
        data: Dictionary matching the configuration schema.

    Returns:
        Validated KasumiConfig instance.

    Raises:
        ValueError: If the configuration fails validation.
    """
    try:
        return KasumiConfig(**data)
    except Exception as e:
        raise ValueError(f"Configuration validation failed: {e}") from e
