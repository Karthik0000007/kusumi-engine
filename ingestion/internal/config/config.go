// Package config provides configuration loading, validation, and environment
// variable override support for the Kasumi Engine ingestion service.
//
// Configuration precedence (highest to lowest):
//  1. Environment variables (KASUMI_* prefix)
//  2. Configuration file (YAML)
//  3. Built-in defaults
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure for the Kasumi Engine.
// It mirrors the shared JSON Schema defined in config/schema.json.
type Config struct {
	Ingestion    IngestionConfig    `yaml:"ingestion" validate:"required"`
	Retrieval    RetrievalConfig    `yaml:"retrieval" validate:"required"`
	Reranking    RerankingConfig    `yaml:"reranking" validate:"required"`
	Redis        RedisConfig        `yaml:"redis" validate:"required"`
	Privacy      PrivacyConfig      `yaml:"privacy" validate:"required"`
	Observability ObservabilityConfig `yaml:"observability"`
}

// IngestionConfig holds settings for the Go gRPC ingestion service.
type IngestionConfig struct {
	Host                  string `yaml:"host" validate:"required" env:"KASUMI_INGESTION_HOST"`
	Port                  int    `yaml:"port" validate:"required,min=1,max=65535" env:"KASUMI_INGESTION_PORT"`
	LogLevel              string `yaml:"log_level" validate:"oneof=debug info warn error" env:"KASUMI_INGESTION_LOG_LEVEL"`
	MaxConcurrentStreams   int    `yaml:"max_concurrent_streams" validate:"min=1,max=10000" env:"KASUMI_INGESTION_MAX_CONCURRENT_STREAMS"`
	BackpressureBufferSize int   `yaml:"backpressure_buffer_size" validate:"min=100,max=1000000" env:"KASUMI_INGESTION_BACKPRESSURE_BUFFER_SIZE"`
	RateLimitRPS          int    `yaml:"rate_limit_rps" validate:"min=1" env:"KASUMI_INGESTION_RATE_LIMIT_RPS"`
	TLSEnabled            bool   `yaml:"tls_enabled" env:"KASUMI_INGESTION_TLS_ENABLED"`
	TLSCertPath           string `yaml:"tls_cert_path" env:"KASUMI_INGESTION_TLS_CERT_PATH"`
	TLSKeyPath            string `yaml:"tls_key_path" env:"KASUMI_INGESTION_TLS_KEY_PATH"`
}

// RetrievalConfig holds settings for the Python FastAPI retrieval service.
type RetrievalConfig struct {
	Host            string `yaml:"host" validate:"required" env:"KASUMI_RETRIEVAL_HOST"`
	Port            int    `yaml:"port" validate:"required,min=1,max=65535" env:"KASUMI_RETRIEVAL_PORT"`
	LogLevel        string `yaml:"log_level" validate:"oneof=debug info warn error" env:"KASUMI_RETRIEVAL_LOG_LEVEL"`
	CandidateCount  int    `yaml:"candidate_count" validate:"min=10,max=5000" env:"KASUMI_RETRIEVAL_CANDIDATE_COUNT"`
	LatencyBudgetMs int    `yaml:"latency_budget_ms" validate:"min=1,max=1000" env:"KASUMI_RETRIEVAL_LATENCY_BUDGET_MS"`
	IndexType       string `yaml:"index_type" validate:"oneof=faiss milvus" env:"KASUMI_RETRIEVAL_INDEX_TYPE"`
	FAISSNList      int    `yaml:"faiss_nlist" validate:"min=1" env:"KASUMI_RETRIEVAL_FAISS_NLIST"`
	FAISSNProbe     int    `yaml:"faiss_nprobe" validate:"min=1" env:"KASUMI_RETRIEVAL_FAISS_NPROBE"`
	EmbeddingDim    int    `yaml:"embedding_dim" validate:"min=1" env:"KASUMI_RETRIEVAL_EMBEDDING_DIM"`
}

// RerankingConfig holds settings for the C++/TensorRT reranking service.
type RerankingConfig struct {
	Host                       string `yaml:"host" validate:"required" env:"KASUMI_RERANKING_HOST"`
	Port                       int    `yaml:"port" validate:"required,min=1,max=65535" env:"KASUMI_RERANKING_PORT"`
	LatencyBudgetMs            int    `yaml:"latency_budget_ms" validate:"min=1,max=1000" env:"KASUMI_RERANKING_LATENCY_BUDGET_MS"`
	ModelType                  string `yaml:"model_type" validate:"oneof=lightgbm two_tower" env:"KASUMI_RERANKING_MODEL_TYPE"`
	TritonURL                  string `yaml:"triton_url" env:"KASUMI_RERANKING_TRITON_URL"`
	FallbackEnabled            bool   `yaml:"fallback_enabled" env:"KASUMI_RERANKING_FALLBACK_ENABLED"`
	DynamicBatchingMaxBatchSize int   `yaml:"dynamic_batching_max_batch_size" validate:"min=1,max=512" env:"KASUMI_RERANKING_DYNAMIC_BATCHING_MAX_BATCH_SIZE"`
}

// RedisConfig holds settings for the Redis feature store.
type RedisConfig struct {
	Host              string `yaml:"host" validate:"required" env:"KASUMI_REDIS_HOST"`
	Port              int    `yaml:"port" validate:"required,min=1,max=65535" env:"KASUMI_REDIS_PORT"`
	Password          string `yaml:"password" env:"KASUMI_REDIS_PASSWORD"`
	DB                int    `yaml:"db" validate:"min=0,max=15" env:"KASUMI_REDIS_DB"`
	MaxConnections    int    `yaml:"max_connections" validate:"min=1,max=1000" env:"KASUMI_REDIS_MAX_CONNECTIONS"`
	KeyPrefix         string `yaml:"key_prefix" env:"KASUMI_REDIS_KEY_PREFIX"`
	FeatureTTLSeconds int    `yaml:"feature_ttl_seconds" validate:"min=60" env:"KASUMI_REDIS_FEATURE_TTL_SECONDS"`
	SessionTTLSeconds int    `yaml:"session_ttl_seconds" validate:"min=60" env:"KASUMI_REDIS_SESSION_TTL_SECONDS"`
}

// PrivacyConfig holds APPI-related privacy settings.
type PrivacyConfig struct {
	Epsilon                  float64 `yaml:"epsilon" validate:"gt=0,max=10" env:"KASUMI_PRIVACY_EPSILON"`
	Sensitivity              float64 `yaml:"sensitivity" validate:"gt=0" env:"KASUMI_PRIVACY_SENSITIVITY"`
	SaltRotationHours        int     `yaml:"salt_rotation_hours" validate:"min=1,max=168" env:"KASUMI_PRIVACY_SALT_ROTATION_HOURS"`
	BudgetCeiling            float64 `yaml:"budget_ceiling" validate:"gt=0" env:"KASUMI_PRIVACY_BUDGET_CEILING"`
	BudgetAlertThreshold     float64 `yaml:"budget_alert_threshold" validate:"gt=0,max=1" env:"KASUMI_PRIVACY_BUDGET_ALERT_THRESHOLD"`
	AggregationWindowMinutes int     `yaml:"aggregation_window_minutes" validate:"min=1,max=1440" env:"KASUMI_PRIVACY_AGGREGATION_WINDOW_MINUTES"`
	RedactSessionIDsInLogs   bool    `yaml:"redact_session_ids_in_logs" env:"KASUMI_PRIVACY_REDACT_SESSION_IDS_IN_LOGS"`
}

// ObservabilityConfig holds monitoring and tracing settings.
type ObservabilityConfig struct {
	MetricsEnabled bool   `yaml:"metrics_enabled" env:"KASUMI_OBSERVABILITY_METRICS_ENABLED"`
	MetricsPort    int    `yaml:"metrics_port" validate:"min=1,max=65535" env:"KASUMI_OBSERVABILITY_METRICS_PORT"`
	TracingEnabled bool   `yaml:"tracing_enabled" env:"KASUMI_OBSERVABILITY_TRACING_ENABLED"`
	LogFormat      string `yaml:"log_format" validate:"oneof=json text" env:"KASUMI_OBSERVABILITY_LOG_FORMAT"`
}

// Defaults returns a Config populated with default values.
func Defaults() *Config {
	return &Config{
		Ingestion: IngestionConfig{
			Host:                  "0.0.0.0",
			Port:                  50051,
			LogLevel:              "info",
			MaxConcurrentStreams:   1000,
			BackpressureBufferSize: 10000,
			RateLimitRPS:          5000,
			TLSEnabled:            false,
		},
		Retrieval: RetrievalConfig{
			Host:            "0.0.0.0",
			Port:            8000,
			LogLevel:        "info",
			CandidateCount:  500,
			LatencyBudgetMs: 20,
			IndexType:       "faiss",
			FAISSNList:      256,
			FAISSNProbe:     16,
			EmbeddingDim:    128,
		},
		Reranking: RerankingConfig{
			Host:                       "0.0.0.0",
			Port:                       8001,
			LatencyBudgetMs:            30,
			ModelType:                  "lightgbm",
			TritonURL:                  "localhost:8002",
			FallbackEnabled:            true,
			DynamicBatchingMaxBatchSize: 32,
		},
		Redis: RedisConfig{
			Host:              "localhost",
			Port:              6379,
			DB:                0,
			MaxConnections:    50,
			KeyPrefix:         "kasumi:",
			FeatureTTLSeconds: 3600,
			SessionTTLSeconds: 1800,
		},
		Privacy: PrivacyConfig{
			Epsilon:                  1.0,
			Sensitivity:              1.0,
			SaltRotationHours:        24,
			BudgetCeiling:            10.0,
			BudgetAlertThreshold:     0.8,
			AggregationWindowMinutes: 15,
			RedactSessionIDsInLogs:   true,
		},
		Observability: ObservabilityConfig{
			MetricsEnabled: true,
			MetricsPort:    9090,
			TracingEnabled: false,
			LogFormat:      "json",
		},
	}
}

// Load reads a YAML configuration file and returns a validated Config.
// It applies the following precedence: env vars > file values > defaults.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	// Read and parse the YAML file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Validate the final configuration
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// LoadFromBytes parses YAML configuration from raw bytes, applies env overrides,
// and validates. Useful for testing.
func LoadFromBytes(data []byte) (*Config, error) {
	cfg := Defaults()

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config data: %w", err)
	}

	applyEnvOverrides(cfg)

	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Validate checks the configuration against structural and business rules.
func Validate(cfg *Config) error {
	v := validator.New()

	if err := v.Struct(cfg); err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return fmt.Errorf("unexpected validation error: %w", err)
		}

		var messages []string
		for _, e := range validationErrors {
			messages = append(messages, formatValidationError(e))
		}
		return fmt.Errorf("configuration validation failed:\n  %s", strings.Join(messages, "\n  "))
	}

	// Business rule: TLS requires both cert and key
	if cfg.Ingestion.TLSEnabled {
		if cfg.Ingestion.TLSCertPath == "" || cfg.Ingestion.TLSKeyPath == "" {
			return fmt.Errorf("configuration validation failed:\n  tls_enabled requires both tls_cert_path and tls_key_path")
		}
	}

	// Business rule: budget_alert_threshold must be less than 1.0 (it's a fraction)
	if cfg.Privacy.BudgetAlertThreshold >= 1.0 {
		return fmt.Errorf("configuration validation failed:\n  privacy.budget_alert_threshold must be less than 1.0, got %f", cfg.Privacy.BudgetAlertThreshold)
	}

	return nil
}

// applyEnvOverrides walks the config struct and applies environment variable
// overrides based on the `env` struct tag.
func applyEnvOverrides(cfg *Config) {
	applyEnvToStruct(reflect.ValueOf(cfg).Elem())
}

func applyEnvToStruct(v reflect.Value) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Recurse into nested structs
		if field.Type.Kind() == reflect.Struct {
			applyEnvToStruct(fieldVal)
			continue
		}

		envKey := field.Tag.Get("env")
		if envKey == "" {
			continue
		}

		envVal, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}

		switch field.Type.Kind() {
		case reflect.String:
			fieldVal.SetString(envVal)
		case reflect.Int:
			if intVal, err := strconv.Atoi(envVal); err == nil {
				fieldVal.SetInt(int64(intVal))
			}
		case reflect.Float64:
			if floatVal, err := strconv.ParseFloat(envVal, 64); err == nil {
				fieldVal.SetFloat(floatVal)
			}
		case reflect.Bool:
			if boolVal, err := strconv.ParseBool(envVal); err == nil {
				fieldVal.SetBool(boolVal)
			}
		}
	}
}

// formatValidationError converts a validator.FieldError into a human-readable message.
func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Namespace())
	case "min":
		return fmt.Sprintf("%s must be at least %s (got %v)", e.Namespace(), e.Param(), e.Value())
	case "max":
		return fmt.Sprintf("%s must be at most %s (got %v)", e.Namespace(), e.Param(), e.Value())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s (got %v)", e.Namespace(), e.Param(), e.Value())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s] (got %v)", e.Namespace(), e.Param(), e.Value())
	default:
		return fmt.Sprintf("%s failed validation: %s", e.Namespace(), e.Tag())
	}
}
