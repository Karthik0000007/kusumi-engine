"""
Kasumi Engine — Structured Logging for the Retrieval Service

structlog-based JSON logging with correlation ID propagation and
session identifier redaction. The log schema matches the Go ingestion
service logger for cross-service consistency:

    {"timestamp": "...", "level": "...", "service": "...", "correlation_id": "...", "event": "..."}
"""

from __future__ import annotations

import logging
import re
import sys
from contextvars import ContextVar
from typing import Any

import structlog

# Context variable for correlation ID propagation across async boundaries.
_correlation_id_var: ContextVar[str] = ContextVar("correlation_id", default="")

# Pattern to match session identifiers for redaction.
# Matches UUIDs, hex strings of 16+ chars, and base64-like tokens
# that follow session-related field names.
_SESSION_ID_PATTERN = re.compile(
    r"(?i)(session[_-]?id|sid|sess)[\"'\s:=]+[\"']?([a-zA-Z0-9+/=_\-]{16,})[\"']?"
)
_REDACTION_REPLACEMENT = r"\1: [REDACTED]"


def set_correlation_id(correlation_id: str) -> None:
    """Set the correlation ID for the current context."""
    _correlation_id_var.set(correlation_id)


def get_correlation_id() -> str:
    """Get the correlation ID from the current context."""
    return _correlation_id_var.get()


def _add_correlation_id(
    logger: Any, method_name: str, event_dict: dict[str, Any]
) -> dict[str, Any]:
    """Structlog processor that adds the correlation ID from context."""
    correlation_id = _correlation_id_var.get()
    if correlation_id:
        event_dict["correlation_id"] = correlation_id
    return event_dict


def _add_service_name(
    service: str,
) -> structlog.types.Processor:
    """Returns a structlog processor that adds the service name."""

    def processor(
        logger: Any, method_name: str, event_dict: dict[str, Any]
    ) -> dict[str, Any]:
        event_dict["service"] = service
        return event_dict

    return processor


def _redact_session_ids(
    logger: Any, method_name: str, event_dict: dict[str, Any]
) -> dict[str, Any]:
    """Structlog processor that redacts session identifiers from all string values."""
    for key, value in event_dict.items():
        if isinstance(value, str):
            event_dict[key] = _SESSION_ID_PATTERN.sub(_REDACTION_REPLACEMENT, value)
    return event_dict


def setup_logging(
    level: str = "info",
    service: str = "retrieval",
    log_format: str = "json",
    redact: bool = True,
) -> None:
    """
    Configure structlog for the Kasumi Engine retrieval service.

    Args:
        level: Log level ("debug", "info", "warn", "error").
        service: Service name for the "service" field.
        log_format: Output format — "json" for production, "text" for dev.
        redact: Whether to redact session identifiers from log output.
    """
    # Map level string to logging constant
    log_level = _parse_level(level)

    # Build the processor chain
    processors: list[structlog.types.Processor] = [
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso", key="timestamp"),
        _add_service_name(service),
        _add_correlation_id,
    ]

    if redact:
        processors.append(_redact_session_ids)

    processors.append(structlog.processors.StackInfoRenderer())
    processors.append(structlog.processors.format_exc_info)

    if log_format == "json":
        processors.append(structlog.processors.JSONRenderer())
    else:
        processors.append(structlog.dev.ConsoleRenderer())

    # Configure structlog
    structlog.configure(
        processors=processors,
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(file=sys.stdout),
        cache_logger_on_first_use=True,
    )

    # Also configure stdlib logging for third-party libraries
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=log_level,
    )


def get_logger(**kwargs: Any) -> structlog.stdlib.BoundLogger:
    """
    Get a structured logger instance.

    Args:
        **kwargs: Additional context to bind to the logger.

    Returns:
        A bound structlog logger.
    """
    return structlog.get_logger(**kwargs)


def _parse_level(level: str) -> int:
    """Convert a string log level to a logging constant."""
    level_map = {
        "debug": logging.DEBUG,
        "info": logging.INFO,
        "warn": logging.WARNING,
        "warning": logging.WARNING,
        "error": logging.ERROR,
    }
    return level_map.get(level.lower(), logging.INFO)
