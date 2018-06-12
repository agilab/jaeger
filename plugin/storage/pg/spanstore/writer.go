package spanstore

import (
	"context"

	"time"

	"fmt"


	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/id_mapping"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/pgutil"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/tables"
	storageMetrics "github.com/jaegertracing/jaeger/storage/spanstore/metrics"
	"github.com/uber/jaeger-client-go/log/zap"
	"github.com/uber/jaeger-lib/metrics"
	"strings"
)

/*
 * 作者:张晓明 时间:18/6/12
 */
type spanWriterMetrics struct {
	spanInsert *storageMetrics.WriteMetrics
}

// SpanWriter is a wrapper around elastic.Client
type SpanWriter struct {
	ctx              context.Context
	db               *pg.DB
	logger           *zap.Logger
	writerMetrics    spanWriterMetrics // TODO: build functions to wrap around each Do fn
	idMappingService id_mapping.IDMappingService
	tableCache       cache.Cache
	modeOption       modeOption
}

func NewSpanWriter(
	db *pg.DB,
	logger *zap.Logger,
	metricsFactory metrics.Factory,
	modeOption modeOption,
) *SpanWriter {
	ctx := context.Background()
	return &SpanWriter{
		ctx:    ctx,
		db:     db,
		logger: logger,
		writerMetrics: spanWriterMetrics{
			spanInsert: storageMetrics.NewWriteMetrics(metricsFactory, "spanCreate"),
		},
		idMappingService: id_mapping.InitAndGetIDMappingService(),
		tableCache: cache.NewLRUWithOptions(
			5,
			&cache.Options{
				TTL: 4 * time.Hour,
			},
		),
		modeOption: modeOption,
	}
}

func (s *SpanWriter) WriteSpan(span *model.Span) error {
	if err := s.createPartitionTable(span.StartTime); err != nil {
		return err
	}

}

func (s *SpanWriter) transportJaegerSpan2PgSpan(span *model.Span) (*tables.Span, error){
	tSpan := new(tables.Span)
	tSpan.SpanID = int64(span.SpanID)
	tSpan.TraceID = int64(span.TraceID.Low)
	tSpan.Request = s.modeOption.getRequestFn(span)
	tSpan.Response = s.modeOption.getResponseFn(span)
	tSpan.UserID = s.modeOption.getUserIDFn(span)
	tSpan.ParentSpanIds = span.ParentSpanIds
	tSpan.StartTime = &span.StartTime
	tSpan.Duration = span.Duration.Nanoseconds()
	tSpan.Tags = buildTags2Map(span.Tags)
	tSpan.Logs = span.Logs
	tSpan.Flags = int64(span.Flags)
	tSpan.Warnings = span.Warnings
	tips := strings.SplitN(span.OperationName, " ", 2)
	var opType, opName string
	if len(tips) == 1 {
		opType = ""
		opName = tips[0]
	}
	if len(tips) == 2 {
		opType = tips[0]
		opType = tips[1]
	}
	op := s.idMappingService.GetOperatorFromName(span.Process.ServiceName, opType, opName)
	tSpan.ServiceID = op.ServiceID
	tSpan.OperatorTypeID = op.OperatorTypeID
	tSpan.OperatorTypeID = op.OperatorTypeID
	tSpan.ParentOperatorIds = span.ParentIds
	tSpan.Process = span.Process
	tSpan.Reference = span.Reference
	return tSpan
}

func buildLogs2Map(tags model.KeyValues) map[string]string {
	logMap := make(map[string]interface{}, 0)
	for _, tag := range tags {
		tagMap[tag.Key] = tag.AsString()
	}
	return tagMap
}

func buildTags2Map(tags model.KeyValues) map[string]interface{} {
	tagMap := make(map[string]interface{}, 0)
	for _, tag := range tags {
		tagMap[tag.Key] = tag.Value()
	}
	return tagMap
}

func (s *SpanWriter) createPartitionTable(startTime time.Time) error {
	tbName := tableNames(startTime)
	if !keyInCache(tbName, s.tableCache) {
		err := pgutil.CreatePartitionSpanTable(startTime)
		if err != nil {
			s.logger.Error(fmt.Sprintf("create index failed:%s", err.Error()))
			return err
		}
		writeCache(tbName, s.tableCache)
	}
	return nil
}
func tableNames(startTime time.Time) string {
	return pgutil.GetStartTimeFmt(startTime)
}
func keyInCache(key string, c cache.Cache) bool {
	return c.Get(key) != nil
}

func writeCache(key string, c cache.Cache) {
	c.Put(key, key)
}
