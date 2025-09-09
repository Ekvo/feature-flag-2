package config

import (
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config - contains url for database, server port with server network, secret key for jwt
type Config struct {
	DB         DataBaseConfig  `envPrefix:"DB_"`
	Cache      CacheConfig     `envPrefix:"CACHE_"`
	Migrations MigrationConfig `envPrefix:"MIGRATION_"`
	Server     ServerConfig    `envPrefix:"SRV_"`
}

// NewConfig - load data from ENV (file or ENV variables)
func NewConfig(pathToEnv string) (*Config, error) {
	log.Print("config: config start")

	cfg := &Config{}
	if err := cfg.parse(pathToEnv); err != nil {
		return nil, fmt.Errorf("config: env.Parse error - {%w};", err)
	}

	log.Print("config: config created")

	return cfg, nil
}

func (cfg *Config) parse(pathToEnv string) error {
	if err := godotenv.Load(pathToEnv); err != nil {
		// work with ENV
		log.Printf("config: .env file error - {%v};", err)
		return err
	}
	if err := env.Parse(cfg); err != nil {
		return err
	}
	cfg.DB.URL = cfg.DB.url()

	log.Print("config: parse end")

	return nil
}

type DataBaseConfig struct {
	Host     string `env:"HOST"`
	Port     uint16 `env:"PORT"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	Name     string `env:"NAME"`
	SSLMode  string `env:"SSLMODE"`

	URL string `env:"-"`

	//MaxConn           uint16        `env:"MAX_CONN"`
	//MinConn           uint16        `env:"MIN_CONN"`
	//ConnMaxLifeTime   time.Duration `env:"CONN_MAX_LIFE_TIME"`
	//ConnMaxIdleTime   time.Duration `env:"CONN_MAX_IDLE_TIME"`
	//ConnTime          time.Duration `env:"CONN_TIMEOUT"`
	//HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD"`
}

func (cfgDB *DataBaseConfig) url() string {
	return fmt.Sprintf(`postgresql://%s:%s@%s:%d/%s`,
		cfgDB.User,
		cfgDB.Password,
		cfgDB.Host,
		cfgDB.Port,
		cfgDB.Name,
	)
}

type CacheConfig struct {
	TTLMiddlewareFiber time.Duration `env:"TTL_MIDDLEWARE_FIBER"`
	TTLLRU             time.Duration `env:"TTL_LRU"`
	SizeLRU            int           `env:"SIZE_LRU"`
}

type MigrationConfig struct {
	PathToMigrations string `env:"PATH"`
	Action           string `env:"ACTION"`
	Version          int64  `env:"VERSION"`
}

type ServerConfig struct {
	Port     string        `env:"PORT"`
	Host     string        `env:"HOST"`
	ShutDown time.Duration `env:"SHUTDOWN"`
}
