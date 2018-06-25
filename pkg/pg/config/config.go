package config

import (
	gopg "github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/pkg/pg"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

type Configuration struct {
	ServerURL string
	Username  string
	Password  string
}

func (c *Configuration) NewClient(logger *zap.Logger, metricsFactory metrics.Factory) (pg.DB, error) {
	db := gopg.Connect(&gopg.Options{
		Addr:     viper.GetString("pg.url"),
		User:     viper.GetString("pg.user"),
		Password: viper.GetString("pg.password"),
		Database: viper.GetString("pg.database"),
	})

	return db, nil
}
