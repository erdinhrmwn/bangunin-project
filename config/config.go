// Package config loads typed application configuration from config.yaml,
// overridable via APP_-prefixed environment variables.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	App        AppConfig
	DB         DBConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Xendit     XenditConfig
	RajaOngkir RajaOngkirConfig
	R2         R2Config
}

type AppConfig struct {
	Name string
	Env  string
	Port int
}

type DBConfig struct {
	DSN         string
	MaxConns    int32
	MinConns    int32
	MaxConnLife time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// XenditConfig, RajaOngkirConfig, R2Config are intentionally sparse — filled
// in when their respective phases land (5 for Xendit/RajaOngkir, 3 for R2).
type XenditConfig struct {
	SecretKey     string
	CallbackToken string
}

type RajaOngkirConfig struct {
	APIKey string
	Mock   bool
}

type R2Config struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Endpoint  string
	UseSSL    bool
}

// requiredKeys are the config keys that must be non-empty for the app to
// start. Fail-fast here beats a cryptic nil-pointer panic three layers deep.
var requiredKeys = []string{"db.dsn", "redis.addr", "jwt.secret"}

// Load reads config.yaml from configPath, applies APP_-prefixed env
// overrides, and fails fast if a required key is missing.
func Load(configPath string) (*Config, error) {
	_ = gotenv.Load() // .env is optional (e.g. absent in prod, using real env vars)

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read %s: %w", configPath, err)
	}

	for _, key := range requiredKeys {
		if v.GetString(key) == "" {
			return nil, fmt.Errorf("config: required key %q is empty", key)
		}
	}

	cfg := &Config{
		App: AppConfig{
			Name: v.GetString("app.name"),
			Env:  v.GetString("app.env"),
			Port: v.GetInt("app.port"),
		},
		DB: DBConfig{
			DSN:         v.GetString("db.dsn"),
			MaxConns:    v.GetInt32("db.max_conns"),
			MinConns:    v.GetInt32("db.min_conns"),
			MaxConnLife: v.GetDuration("db.max_conn_life"),
		},
		Redis: RedisConfig{
			Addr:     v.GetString("redis.addr"),
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		JWT: JWTConfig{
			Secret:     v.GetString("jwt.secret"),
			AccessTTL:  v.GetDuration("jwt.access_ttl"),
			RefreshTTL: v.GetDuration("jwt.refresh_ttl"),
		},
		Xendit: XenditConfig{
			SecretKey:     v.GetString("xendit.secret_key"),
			CallbackToken: v.GetString("xendit.callback_token"),
		},
		RajaOngkir: RajaOngkirConfig{
			APIKey: v.GetString("rajaongkir.api_key"),
			Mock:   v.GetBool("rajaongkir.mock"),
		},
		R2: R2Config{
			AccessKey: v.GetString("r2.access_key"),
			SecretKey: v.GetString("r2.secret_key"),
			Bucket:    v.GetString("r2.bucket"),
			Endpoint:  v.GetString("r2.endpoint"),
			UseSSL:    v.GetBool("r2.use_ssl"),
		},
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "bangunin")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("db.max_conns", 10)
	v.SetDefault("db.min_conns", 2)
	v.SetDefault("db.max_conn_life", "1h")
	v.SetDefault("redis.db", 0)
	v.SetDefault("jwt.access_ttl", "15m")
	v.SetDefault("jwt.refresh_ttl", "168h")
	v.SetDefault("rajaongkir.mock", true)
}
