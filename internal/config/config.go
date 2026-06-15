package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB     string

	JWTSecret string

	SeetLock time.Duration

	CleanupInterval time.Duration
}

func LoadConfig(path string) *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("failed to read config: ", err)
	}

	return &Config{
		AppPort: viper.GetString("app.port"),

		DBHost:     viper.GetString("database.host"),
		DBPort:     viper.GetString("database.port"),
		DBUser:     viper.GetString("database.user"),
		DBPassword: viper.GetString("database.password"),
		DBName:     viper.GetString("database.name"),
		DBSSLMode:  viper.GetString("database.sslmode"),

		RedisHost:     viper.GetString("redis.host"),
		RedisPort:     viper.GetString("redis.port"),
		RedisPassword: viper.GetString("redis.password"),
		RedisDB:     viper.GetString("redis.db"),

		JWTSecret: viper.GetString("jwt.secret"),

		SeetLock:        viper.GetDuration("seetlock.time"),
		CleanupInterval: viper.GetDuration("cleanupinterval.time"),
	}
}
