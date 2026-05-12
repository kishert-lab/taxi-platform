package configs

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Host string
		Port int
	}

	Database struct {
		Host     string
		Port     int
		User     string
		Password string
		DBName   string
		SSLMode  string
	}

	Redis struct {
		Host string
		Port int
	}

	JWT struct {
		Secret    string
		AccessTTL string
	}
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var cfg Config

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}