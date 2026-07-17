package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Defaults()
	
	if err := Validate(cfg); err != nil {
		t.Fatalf("default configuration is invalid: %v", err)
	}

	if cfg.Ingestion.Port != 50051 {
		t.Errorf("expected default ingestion port to be 50051, got %d", cfg.Ingestion.Port)
	}
	
	if cfg.Privacy.Epsilon != 1.0 {
		t.Errorf("expected default epsilon to be 1.0, got %f", cfg.Privacy.Epsilon)
	}
}

func TestLoadFromYAML(t *testing.T) {
	yamlContent := `
ingestion:
  host: "127.0.0.1"
  port: 9999
  log_level: "debug"
  max_concurrent_streams: 100
  backpressure_buffer_size: 200
  rate_limit_rps: 50
  tls_enabled: false
retrieval:
  host: "0.0.0.0"
  port: 8000
  log_level: "info"
  candidate_count: 500
  latency_budget_ms: 20
  index_type: "faiss"
  faiss_nlist: 256
  faiss_nprobe: 16
  embedding_dim: 128
reranking:
  host: "0.0.0.0"
  port: 8001
  latency_budget_ms: 30
  model_type: "lightgbm"
  triton_url: "localhost:8002"
  fallback_enabled: true
  dynamic_batching_max_batch_size: 32
redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  max_connections: 50
  key_prefix: "test:"
  feature_ttl_seconds: 3600
  session_ttl_seconds: 1800
privacy:
  epsilon: 2.5
  sensitivity: 1.0
  salt_rotation_hours: 12
  budget_ceiling: 20.0
  budget_alert_threshold: 0.9
  aggregation_window_minutes: 10
  redact_session_ids_in_logs: true
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test yaml file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Ingestion.Port != 9999 {
		t.Errorf("expected ingestion port 9999, got %d", cfg.Ingestion.Port)
	}
	if cfg.Privacy.Epsilon != 2.5 {
		t.Errorf("expected epsilon 2.5, got %f", cfg.Privacy.Epsilon)
	}
}

func TestEnvOverrides(t *testing.T) {
	// Set env vars
	os.Setenv("KASUMI_INGESTION_PORT", "8888")
	os.Setenv("KASUMI_PRIVACY_EPSILON", "5.0")
	os.Setenv("KASUMI_INGESTION_TLS_ENABLED", "true")
	os.Setenv("KASUMI_INGESTION_TLS_CERT_PATH", "/tmp/cert")
	os.Setenv("KASUMI_INGESTION_TLS_KEY_PATH", "/tmp/key")
	
	defer func() {
		os.Unsetenv("KASUMI_INGESTION_PORT")
		os.Unsetenv("KASUMI_PRIVACY_EPSILON")
		os.Unsetenv("KASUMI_INGESTION_TLS_ENABLED")
		os.Unsetenv("KASUMI_INGESTION_TLS_CERT_PATH")
		os.Unsetenv("KASUMI_INGESTION_TLS_KEY_PATH")
	}()

	cfg := Defaults()
	applyEnvOverrides(cfg)

	if cfg.Ingestion.Port != 8888 {
		t.Errorf("expected env overridden port 8888, got %d", cfg.Ingestion.Port)
	}
	
	if cfg.Privacy.Epsilon != 5.0 {
		t.Errorf("expected env overridden epsilon 5.0, got %f", cfg.Privacy.Epsilon)
	}
	
	if !cfg.Ingestion.TLSEnabled {
		t.Errorf("expected env overridden tls_enabled true, got false")
	}
}

func TestValidationFails(t *testing.T) {
	cfg := Defaults()
	
	// Test TLS without cert
	cfg.Ingestion.TLSEnabled = true
	cfg.Ingestion.TLSCertPath = ""
	
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation to fail when TLS enabled but no cert path provided")
	}
	
	// Reset
	cfg = Defaults()
	
	// Test out of bounds budget alert threshold
	cfg.Privacy.BudgetAlertThreshold = 1.5
	
	err = Validate(cfg)
	if err == nil {
		t.Error("expected validation to fail when budget alert threshold >= 1.0")
	}
}
