import pytest
from pydantic import ValidationError

from retrieval.config import KasumiConfig, load_config_from_dict

def test_default_config():
    cfg = KasumiConfig()

    assert cfg.ingestion.port == 50051
    assert cfg.privacy.epsilon == 1.0
    assert cfg.retrieval.candidate_count == 500

def test_env_override(monkeypatch):
    monkeypatch.setenv("KASUMI_INGESTION__PORT", "8888")
    monkeypatch.setenv("KASUMI_PRIVACY__EPSILON", "5.0")
    monkeypatch.setenv("KASUMI_INGESTION__TLS_ENABLED", "true")
    monkeypatch.setenv("KASUMI_INGESTION__TLS_CERT_PATH", "/tmp/cert")
    monkeypatch.setenv("KASUMI_INGESTION__TLS_KEY_PATH", "/tmp/key")

    cfg = KasumiConfig()

    assert cfg.ingestion.port == 8888
    assert cfg.privacy.epsilon == 5.0
    assert cfg.ingestion.tls_enabled is True

def test_validation_fails_tls():
    with pytest.raises(ValidationError):
        KasumiConfig(
            ingestion={
                "tls_enabled": True,
                "tls_cert_path": "",
                "tls_key_path": ""
            }
        )

def test_validation_fails_budget_threshold():
    with pytest.raises(ValidationError):
        KasumiConfig(
            privacy={
                "budget_alert_threshold": 1.5
            }
        )

def test_load_from_dict():
    data = {
        "ingestion": {
            "port": 9999,
            "host": "localhost"
        },
        "privacy": {
            "epsilon": 2.5
        }
    }

    cfg = load_config_from_dict(data)
    assert cfg.ingestion.port == 9999
    assert cfg.privacy.epsilon == 2.5
