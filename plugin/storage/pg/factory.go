package pg

import (
	"flag"
	"time"

	gopg "github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/pkg/pg"
	"github.com/jaegertracing/jaeger/pkg/testutils"
	pgDepStore "github.com/jaegertracing/jaeger/plugin/storage/pg/dependencystore"
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
	Options        *Options
	db             pg.DB
	metricsFactory metrics.Factory
	logger         *zap.Logger
}

func NewFactory() *Factory {
	return &Factory{
		Options: NewOption("pg"),
	}
}

func (f *Factory) AddFlags(flagSet *flag.FlagSet) {
	f.Options.AddFlags(flagSet)
}

func (f *Factory) InitFromViper(v *viper.Viper) {
	f.Options.InitFromViper(v)

}

func (f *Factory) Initialize(metricsFactory metrics.Factory, logger *zap.Logger) error {
	f.metricsFactory = metricsFactory
	f.logger = logger
	db, err := f.Options.primary.NewClient(logger, metricsFactory)
	if err != nil {
		return err
	}
	f.db = db
	return nil
}

func (f *Factory) CreateSpanReader() (spanstore.Reader, error) {
	db := f.db.(*gopg.DB)
	spanReader := pgSpanStore.NewSpanReader(db, f.logger, 0, f.metricsFactory)

	return spanReader, nil
}

func (f *Factory) CreateSpanWriter() (spanstore.Writer, error) {
	logger, _ := testutils.NewLogger()
	db := f.db.(*gopg.DB)
	spanWriter := pgSpanStore.NewSpanWriter(db, logger, f.metricsFactory, pgSpanStore.WriterOption{
		RequestLogKey:       "request",
		ResponseLogKey:      "response",
		UserIdTagKey:        "user.id",
		MaxBatchLen:         5000,
		BufferFlushInterval: time.Second,
	})
	return spanWriter, nil
}

func (f *Factory) CreateDependencyReader() (dependencystore.Reader, error) {
	db := f.db.(*gopg.DB)
	return pgDepStore.NewDependencyStore(db, f.logger), nil
}
