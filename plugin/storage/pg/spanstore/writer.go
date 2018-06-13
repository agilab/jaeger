package spanstore

import (
	"context"

	"time"

	"fmt"

	"strings"

	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/id_mapping"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/pgutil"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/tables"
	storageMetrics "github.com/jaegertracing/jaeger/storage/spanstore/metrics"
	"github.com/uber/jaeger-client-go/log/zap"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/atomic"
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
	option           writerOption
	spanArrayBuf     []*tables.Span
	arrayBufLen      *atomic.Int64
}

func NewSpanWriter(
	db *pg.DB,
	logger *zap.Logger,
	metricsFactory metrics.Factory,
	modeOption writerOption,
) *SpanWriter {
	ctx := context.Background()
	s := &SpanWriter{
		ctx:    ctx,
		db:     db,
		logger: logger,
		writerMetrics: spanWriterMetrics{
			spanInsert: storageMetrics.NewWriteMetrics(metricsFactory, "spanCreate"),
		},
		idMappingService: id_mapping.InitAndGetIDMappingService(db),
		tableCache: cache.NewLRUWithOptions(
			5,
			&cache.Options{
				TTL: 4 * time.Hour,
			},
		),
		option:       modeOption,
		spanArrayBuf: make([]*tables.Span, 0, modeOption.maxBatchLen),
	}
	go s.processQueue()
	return s
}

func (s *SpanWriter) processQueue() {
	timer := time.NewTicker(s.option.bufferFlushInterval)
	for {
		select {
		case <-timer.C:
			s.Flush()
		}
	}
}

func (s *SpanWriter) WriteSpan(span *model.Span) error {
	var err error
	if err = s.createPartitionTable(span.StartTime); err != nil {
		return err
	}
	var tSpan *tables.Span
	if tSpan, err = s.transportJaegerSpan2PgSpan(span); err != nil {
		return err
	}
	s.spanArrayBuf = append(s.spanArrayBuf, tSpan)
	if len(s.spanArrayBuf) >= s.option.maxBatchLen {
		return s.Flush()
	}
	return nil
}

func (s *SpanWriter) transportJaegerSpan2PgSpan(span *model.Span) (*tables.Span, error) {
	tSpan := new(tables.Span)
	tSpan.SpanID = int64(span.SpanID)
	tSpan.TraceID = int64(span.TraceID.Low)
	tSpan.Request = s.option.getRequestFn(span)
	tSpan.Response = s.option.getResponseFn(span)
	tSpan.UserID = s.option.getUserIDFn(span)
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
	tSpan.ServiceID = s.idMappingService.GetIDFromName(span.Process.ServiceName, tables.OpMetaTypeEnum.Service)
	tSpan.OperatorTypeID = s.idMappingService.GetIDFromName(opType, tables.OpMetaTypeEnum.OpType)
	tSpan.OperatorTypeID = s.idMappingService.GetIDFromName(opName, tables.OpMetaTypeEnum.OpName)
	tSpan.ParentOperatorIds = make([]int64, 0)
	for _, parentOperatorName := range span.ParentOperatorNames {
		opId := s.idMappingService.GetIDFromName(parentOperatorName, tables.OpMetaTypeEnum.OpName)
		tSpan.ParentOperatorIds = append(tSpan.ParentOperatorIds, opId)
	}
	tSpan.Process = buildTags2Map(span.Process.Tags)
	tSpan.Reference = span.References
	return tSpan, nil
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
		err := pgutil.CreatePartitionSpanTable(s.db, startTime)
		if err != nil {
			s.logger.Error(fmt.Sprintf("create index failed:%s", err.Error()))
			return err
		}
		writeCache(tbName, s.tableCache)
	}
	return nil
}
func (s *SpanWriter) Flush() error {
	start := time.Now()
	count := len(s.spanArrayBuf)
	if count == 0 {
		return nil
	}
	_, err := s.db.Model(&tables.Span{}).Insert(s.spanArrayBuf)
	s.spanArrayBuf = s.spanArrayBuf[:0]
	s.writerMetrics.spanInsert.Emit(err, time.Since(start))
	return err
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
