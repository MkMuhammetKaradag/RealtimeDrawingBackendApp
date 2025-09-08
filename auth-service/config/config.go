package config

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Postgres PostgresConfig `mapstructure:"postgres"`
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

func Read() Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/app")
	viper.AddConfigPath("/")

	if err := viper.ReadInConfig(); err != nil {
		//log.Fatalf("FAİLED TO READ CONFİGURATİON FİLE: %v", err)
		zap.L().Error("FAİLED TO READ CONFİGURATİON FİLE", zap.Error(err))

	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		//log.Fatalf("Configuration could not be parsed: %v", err)
		zap.L().Error("Configuration could not be parsed", zap.Error(err))
	}

	return config
}
