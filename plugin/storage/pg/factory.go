package pg

import (
	"flag"
	"time"

	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/pkg/testutils"
	pgSpanStore "github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

/*
 * 作者:张晓明 时间:18/6/19
 */

type FactoryOption struct {
	RequestLogKey       string
	ResponseLogKey      string
	UserIdTagKey        string
	MaxBatchLen         int           //向数据库里面批量插入的最多数据条数
	BufferFlushInterval time.Duration //数据等待的最长时间
}

// Factory implements storage.Factory for Elasticsearch backend.
type Factory struct {
	db             *pg.DB
	metricsFactory metrics.Factory
	logger         *zap.Logger
}

func (f Factory) AddFlags(flagSet *flag.FlagSet) {
	panic("implement me")
}

func (f Factory) InitFromViper(v *viper.Viper) {
	panic("implement me")
}

func (f Factory) Initialize(metricsFactory metrics.Factory, logger *zap.Logger) error {
	f.metricsFactory = metricsFactory
	f.logger = logger
	return nil
}

func (Factory) CreateSpanReader() (spanstore.Reader, error) {
	panic("implement me")
}

func (f Factory) CreateSpanWriter() (spanstore.Writer, error) {
	logger, _ := testutils.NewLogger()
	spanWriter := pgSpanStore.NewSpanWriter(f.db, logger, f.metricsFactory, pgSpanStore.WriterOption{
		RequestLogKey:       "request",
		ResponseLogKey:      "response",
		UserIdTagKey:        "user.id",
		MaxBatchLen:         5000,
		BufferFlushInterval: time.Second,
	})
	return spanWriter, nil
}

func (Factory) CreateDependencyReader() (dependencystore.Reader, error) {
	panic("implement me")
}
