package config

import (
	"log"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type DBPoolConfig struct {
	MaxConns          int32         `env:"DB_MAX_CONNS" envDefault:"10"`
	MinConns          int32         `env:"DB_MIN_CONNS" envDefault:"2"`
	MaxConnLifetime   time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"1h"`
	MaxConnIdleTime   time.Duration `env:"DB_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	HealthCheckPeriod time.Duration `env:"DB_HEALTH_CHECK_PERIOD" envDefault:"5m"`
	ConnectTimeout    time.Duration `env:"DB_CONNECT_TIMEOUT" envDefault:"10s"`
}

type Config struct {
	// server
	AppPort string `env:"APP_PORT" envDefault:"8080"`

	// database
	DatabaseURL string `env:"DATABASE_URL,required"`
	DBPool      DBPoolConfig

	// redis / asynq
	RedisAddr     string `env:"REDIS_ADDR,required"`
	RedisUsername string `env:"REDIS_USERNAME,required"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`
	RedisUrl      string `env:"REDIS_URL,required"`
}

var (
	cfg  *Config
	once sync.Once
)

// Load initializes configuration and FAILS FAST if anything is wrong.
func Load() *Config {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("no .env file found (using system environment)")
		}

		c := Config{}

		if err := env.Parse(&c); err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		cfg = &c
	})

	return cfg
}
