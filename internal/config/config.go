package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type AppConfig struct {
	Port string `env:"PORT" envDefault:"8080"`
}

type DBConfig struct {
	DBDriver string `env:"DB_DRIVER" envDefault:"postgres"`
	DBSource string `env:"DB_SOURCE" envDefault:"postgresql://user:password@localhost:5432/dbname?sslmode=disable"`
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR"     envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB"       envDefault:"0"`
}

type GatewayConfig struct {
	WorkerPoolSize     int           `env:"WORKER_POOL_SIZE"      envDefault:"500"`
	RequestTimeout     time.Duration `env:"REQUEST_TIMEOUT"       envDefault:"60s"`
	SessionTTL         time.Duration `env:"SESSION_TTL"           envDefault:"2h"`
	DefaultDailyBudget int           `env:"DEFAULT_DAILY_BUDGET"  envDefault:"100000"`
}

type ProviderConfig struct {
	// Comma-separated API keys for key rotation
	OpenAIAPIKeys    string `env:"OPENAI_API_KEYS"`
	AnthropicAPIKeys string `env:"ANTHROPIC_API_KEYS"`
	// Legacy single-key support (deprecated, use comma-separated keys)
	OpenAIKey      string `env:"OPENAI_API_KEY"`
	AnthropicKey   string `env:"ANTHROPIC_API_KEY"`
	InternalAPIURL string `env:"INTERNAL_MODEL_URL" envDefault:"http://internal-llm:8000"`
}

// GetOpenAIKeys returns all configured OpenAI API keys
func (c *ProviderConfig) GetOpenAIKeys() []string {
	return parseKeys(c.OpenAIAPIKeys, c.OpenAIKey)
}

// GetAnthropicKeys returns all configured Anthropic API keys
func (c *ProviderConfig) GetAnthropicKeys() []string {
	return parseKeys(c.AnthropicAPIKeys, c.AnthropicKey)
}

// parseKeys parses comma-separated keys and falls back to single key
func parseKeys(multiKey, singleKey string) []string {
	var keys []string

	// Parse comma-separated keys
	if multiKey != "" {
		for _, k := range strings.Split(multiKey, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
	}

	// Fall back to single key if no multi-keys configured
	if len(keys) == 0 && singleKey != "" {
		keys = append(keys, singleKey)
	}

	return keys
}

type Config struct {
	AppConfig      `json:"app"`
	DBConfig       `json:"db"`
	RedisConfig    `json:"redis"`
	GatewayConfig  `json:"gateway"`
	ProviderConfig `json:"provider"`
}

func NewConfig() (*Config, error) {
	_ = godotenv.Load()
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
