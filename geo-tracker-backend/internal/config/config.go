package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log_level"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type BrandConfig struct {
	Name string `mapstructure:"name"`
}

type ProviderConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	APIKey         string `mapstructure:"api_key"`
	ProbeModel     string `mapstructure:"probe_model"`
	ExtractModel   string `mapstructure:"extract_model"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

type ProvidersConfig struct {
	Claude     ProviderConfig `mapstructure:"claude"`
	ChatGPT    ProviderConfig `mapstructure:"chatgpt"`
	Perplexity ProviderConfig `mapstructure:"perplexity"`
	Gemini     ProviderConfig `mapstructure:"gemini"`
}

type RunnerConfig struct {
	Workers            int `mapstructure:"workers"`
	SamplesPerPrompt   int `mapstructure:"samples_per_prompt"`
	RetryAttempts      int `mapstructure:"retry_attempts"`
	RetryDelaySeconds  int `mapstructure:"retry_delay_seconds"`
	RateLimitPerMinute int `mapstructure:"rate_limit_per_minute"`
}

type CostRate struct {
	Input  float64 `mapstructure:"input"`
	Output float64 `mapstructure:"output"`
}

type CostRatesConfig struct {
	ClaudeSonnet CostRate `mapstructure:"claude_sonnet"`
	ClaudeHaiku  CostRate `mapstructure:"claude_haiku"`
	GPT4o        CostRate `mapstructure:"gpt4o"`
	GPT4oMini    CostRate `mapstructure:"gpt4o_mini"`
	Perplexity   CostRate `mapstructure:"perplexity"`
	GeminiFlash  CostRate `mapstructure:"gemini_flash"`
}

type ServeConfig struct {
	Host        string   `mapstructure:"host"`
	Port        int      `mapstructure:"port"`
	CORSOrigins []string `mapstructure:"cors_origins"`
}

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Brands    []BrandConfig   `mapstructure:"brands"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Runner    RunnerConfig    `mapstructure:"runner"`
	CostRates CostRatesConfig `mapstructure:"cost_rates"`
	Serve     ServeConfig     `mapstructure:"serve"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Env vars: GEOTRACKER_DATABASE_PASSWORD
	viper.SetEnvPrefix("GEOTRACKER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	// Override with local config if exists
	viper.SetConfigName("config.local")
	if err := viper.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			// No-op: config.local is optional
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	var errs []string

	if c.Database.Host == "" {
		errs = append(errs, "database.host is required")
	}
	if c.Database.Name == "" {
		errs = append(errs, "database.name is required")
	}

	if len(c.Brands) == 0 {
		errs = append(errs, "at least one brand must be configured")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}
