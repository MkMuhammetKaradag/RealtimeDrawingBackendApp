package config

import (
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Server       ServerConfig       `mapstructure:"server"`
	Postgres     PostgresConfig     `mapstructure:"postgres"`
	SessionRedis SessionRedisConfig `mapstructure:"sessionredis"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type ServerConfig struct {
	Port        string `mapstructure:"port"`
	Host        string `mapstructure:"host"`
	Description string `mapstructure:"description"`
}

type PostgresConfig struct {
	Port     string `mapstructure:"port"`
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

type SessionRedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func Read() Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/app")
	viper.AddConfigPath("/")

	// Defaults
	viper.SetDefault("server.port", "8081")
	viper.SetDefault("server.host", "0.0.0.0")

	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.user", "myuser")
	viper.SetDefault("postgres.password", "mypassword")
	viper.SetDefault("postgres.db", "authdb")

	// ENV overrides with prefix AUTH_ and dot-to-underscore replacement
	viper.SetEnvPrefix("AUTH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		zap.L().Warn("Failed to read configuration file", zap.Error(err))
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		zap.L().Error("Configuration could not be parsed", zap.Error(err))
	}

	return config
}
