package spanstore

import (
	"context"

	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/tables"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

/*
 * 作者:张晓明 时间:18/6/13
 */

type SpanReader struct {
	ctx context.Context
	db  *pg.DB
}

func (s SpanReader) GetTrace(traceID model.TraceID) (*model.Trace, error) {
	tSpans := make([]tables.Span, 0)
	err := s.db.Model(&tSpans).Where("trace_id = ?", traceID.Low).Select(tSpans)
	if err != nil {
		return nil, err
	}
	for _, tSpan := range tSpans {
		span := transportPgSpan2JaegerSpan(tSpan)
	}
}
func transportPgSpan2JaegerSpan(tSpan tables.Span) *model.Span {
	span := new(model.Span)
	span.SpanID = model.SpanID(tSpan.SpanID)
	span.TraceID = model.TraceID{High: tSpan.TraceIDHigh, Low: tSpan.TraceIDLow}
	span.ParentSpanIds = tSpan.ParentSpanIds
	span.StartTime = *tSpan.StartTime
	span.Duration = time.Duration(tSpan.Duration)
	span.Tags = buildMap2Tags(tSpan.Tags)
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
func buildMap2Tags(m map[string]interface{}) model.KeyValues {
	tags := make([]model.KeyValues, len(m))
	for k, v := range m {
		var tag model.KeyValue
		if vStr, ok := v.(string); ok {
			tag = model.String(k, vStr)
		} else if vBool, ok := v.(bool); ok {
			tag = model.Bool(k, vBool)
		}
	}
}

func (SpanReader) GetServices() ([]string, error) {
	panic("implement me")
}

func (SpanReader) GetOperations(service string) ([]string, error) {
	panic("implement me")
}

func (SpanReader) FindTraces(query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	panic("implement me")
}
