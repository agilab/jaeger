package config

import (
	gopg "github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/pkg/pg"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

type Configuration struct {
	ServerURL string
	Username  string
	Password  string
	Database  string
}

func (c *Configuration) NewClient(logger *zap.Logger, metricsFactory metrics.Factory) (pg.DB, error) {
	db := gopg.Connect(&gopg.Options{
		Addr:     c.ServerURL,
		User:     c.Username,
		Password: c.Password,
		Database: c.Database,
	})

	return db, nil
}
