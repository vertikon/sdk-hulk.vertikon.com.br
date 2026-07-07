package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	NATS     NATSConfig     `mapstructure:"nats"`
	AI       AIConfig       `mapstructure:"ai"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Meta     MetaConfig     `mapstructure:"meta"`
}

type MetaConfig struct {
	APIURL string `mapstructure:"api_url"`
	Token  string `mapstructure:"token"`
}

type AppConfig struct {
	Name        string     `mapstructure:"name"`
	Environment string     `mapstructure:"environment"`
	LogLevel    string     `mapstructure:"log_level"`
	HTTP        HTTPConfig `mapstructure:"http"`
}

type HTTPConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type NATSConfig struct {
	URL string `mapstructure:"url"`
}

type AIConfig struct {
	Provider string `mapstructure:"provider"` // openai, gemini, anthropic
	APIKey   string `mapstructure:"api_key"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("app.name", "vertikon-monolith")
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.http.port", 8080)
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("redis.url", "redis://localhost:6379")

	// Env Vars
	v.SetEnvPrefix("VERTIKON")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Compat: aceita env vars sem prefixo (usadas em docker/ops) e com prefixo.
	// Observação: BindEnv com múltiplos nomes permite migração incremental.
	_ = v.BindEnv("app.environment", "APP_ENVIRONMENT", "VERTIKON_APP_ENVIRONMENT")
	_ = v.BindEnv("app.log_level", "LOG_LEVEL", "VERTIKON_APP_LOG_LEVEL", "VERTIKON_LOG_LEVEL")
	_ = v.BindEnv("app.http.port", "HTTP_PORT", "VERTIKON_APP_HTTP_PORT", "VERTIKON_HTTP_PORT")

	_ = v.BindEnv("database.host", "DATABASE_HOST", "VERTIKON_DATABASE_HOST")
	_ = v.BindEnv("database.port", "DATABASE_PORT", "VERTIKON_DATABASE_PORT")
	_ = v.BindEnv("database.user", "DATABASE_USER", "VERTIKON_DATABASE_USER")
	_ = v.BindEnv("database.password", "DATABASE_PASSWORD", "VERTIKON_DATABASE_PASSWORD")
	_ = v.BindEnv("database.dbname", "DATABASE_DBNAME", "VERTIKON_DATABASE_DBNAME")
	_ = v.BindEnv("database.sslmode", "DATABASE_SSLMODE", "VERTIKON_DATABASE_SSLMODE")

	_ = v.BindEnv("nats.url", "NATS_URL", "VERTIKON_NATS_URL")
	_ = v.BindEnv("redis.url", "REDIS_URL", "REDIS_ADDR", "VERTIKON_REDIS_URL")

	_ = v.BindEnv("ai.provider", "AI_PROVIDER", "VERTIKON_AI_PROVIDER")
	_ = v.BindEnv("ai.api_key", "AI_API_KEY", "OPENAI_API_KEY", "VERTIKON_AI_API_KEY")

	_ = v.BindEnv("meta.api_url", "META_API_URL", "VERTIKON_META_API_URL")
	_ = v.BindEnv("meta.token", "META_TOKEN", "VERTIKON_META_TOKEN")

	// Config File
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found is okay if env vars are set
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
