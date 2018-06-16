package spanstore

import (
	"context"

	"time"

	"strings"

	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/id_mapping"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/tables"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

/*
 * 作者:张晓明 时间:18/6/13
 */

type SpanReader struct {
	ctx              context.Context
	db               *pg.DB
	idMappingService id_mapping.IDMappingService
}

func (s SpanReader) GetTrace(traceID model.TraceID) (*model.Trace, error) {
	tSpans := make([]*tables.Span, 0)
	err := s.db.Model(&tSpans).Where("trace_id = ?", traceID.Low).Select(tSpans)
	if err != nil {
		return nil, err
	}
	spans := make([]*model.Span, 0, len(tSpans))
	for _, tSpan := range tSpans {
		span := s.transportPgSpan2JaegerSpan(tSpan)
		spans = append(spans, span)
	}
	return &model.Trace{
		Spans:    spans,
		Warnings: []string{},
	}, nil
}
func (s SpanReader) transportPgSpan2JaegerSpan(tSpan *tables.Span) *model.Span {
	span := new(model.Span)
	span.SpanID = model.SpanID(tSpan.SpanID)
	span.TraceID = model.TraceID{High: tSpan.TraceIDHigh, Low: tSpan.TraceIDLow}
	span.ParentSpanIds = tSpan.ParentSpanIds
	span.StartTime = *tSpan.StartTime
	span.Duration = time.Duration(tSpan.Duration)
	span.Tags = buildMap2Tags(tSpan.Tags)
	span.Logs = tSpan.Logs
	span.Flags = model.Flags(tSpan.Flags)
	span.Warnings = tSpan.Warnings
	span.OperationName = s.idMappingService.GetNameFromId(tSpan.OperatorTypeID, tables.OpMetaTypeEnum.OpType) +
		" " + s.idMappingService.GetNameFromId(tSpan.OperatorID, tables.OpMetaTypeEnum.OpName)
	span.Process = s.buildModelProcess(tSpan.ServiceID, tSpan.Process)
	return span
}

func (s SpanReader) buildModelProcess(serviceID int64, processMap map[string]interface{}) *model.Process {
	process := model.Process{
		ServiceName: s.idMappingService.GetNameFromId(serviceID, tables.OpMetaTypeEnum.Service),
		Tags:        buildMap2Tags(processMap),
	}
	return &process
}

func buildMap2Tags(m map[string]interface{}) model.KeyValues {
	tags := model.KeyValues{}
	for k, v := range m {
		var tag model.KeyValue
		if vStr, ok := v.(string); ok {
			tag = model.String(k, vStr)
		} else if vBool, ok := v.(bool); ok {
			tag = model.Bool(k, vBool)
		} else if vInt64, ok := v.(int64); ok {
			tag = model.Int64(k, vInt64)
		} else if vInt, ok := v.(int); ok {
			tag = model.Int64(k, int64(vInt))
		} else if vFloat64, ok := v.(float64); ok {
			tag = model.Float64(k, vFloat64)
		} else if vFloat32, ok := v.(float32); ok {
			tag = model.Float64(k, float64(vFloat32))
		} else if vBytes, ok := v.([]byte); ok {
			tag = model.Binary(k, vBytes)
		}
		tags = append(tags, tag)
	}
	return tags
}

func (s *SpanReader) GetServices() ([]string, error) {
	metas := make([]*tables.OpMeta, 0)
	err := s.db.Model(&metas).Where("type = ?", tables.OpMetaTypeEnum.Service).Select()
	if err != nil {
		return nil, err
	}
	serviceNames := make([]string, 0, len(metas))
	for _, meta := range metas {
		serviceNames = append(serviceNames, meta.Name)
	}
	return serviceNames, nil
}

func (s *SpanReader) GetOperations(service string) ([]string, error) {
	serviceID := s.idMappingService.GetIdFromName(service, tables.OpMetaTypeEnum.Service)
	opRelations := make([]*tables.OpRelation, 0)
	err := s.db.Model(&opRelations).Where("service_id = ?", serviceID).Select()
	if err != nil {
		return nil, err
	}
	ops := make([]string, 0, len(opRelations))
	for _, r := range opRelations {
		opFullName := s.idMappingService.GetNameFromId(r.OpTypeId, tables.OpMetaTypeEnum.OpType) +
			s.idMappingService.GetNameFromId(r.OpNameId, tables.OpMetaTypeEnum.OpName)
		ops = append(ops, opFullName)
	}
	return ops, nil
}

func (s SpanReader) FindTraces(query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	var opTypeId, opNameId, serviceID int64
	if query.ServiceName != "" {
		serviceID = s.idMappingService.GetIdFromName(query.ServiceName, tables.OpMetaTypeEnum.Service)
	}
	var opType, opName = "", ""
	if query.OperationName != "" {
		tips := strings.SplitN(query.OperationName, " ", 2)
		if len(tips) == 1 {
			opType = ""
			opName = tips[0]
			opNameId = s.idMappingService.GetIdFromName(opName, tables.OpMetaTypeEnum.OpName)
		}
		if len(tips) == 2 {
			opType = tips[0]
			opName = tips[1]
			opTypeId = s.idMappingService.GetIdFromName(opType, tables.OpMetaTypeEnum.OpType)
			opNameId = s.idMappingService.GetIdFromName(opName, tables.OpMetaTypeEnum.OpName)
		}
	}
	var traceIds []struct {
		TraceIDLow  uint64
		TraceIDHigh uint64
	}
	dbq := s.db.Model((*tables.Span)(nil)).Column("trace_id_low,trace_id_high")
	if serviceID != 0 {
		dbq = dbq.Where("service_id = ?", serviceID)
	}
	if opTypeId != 0 {
		dbq = dbq.Where("operator_type_id = ?", opType)
	}
	if opNameId != 0 {
		dbq = dbq.Where("operator_id = ?", opType)
	}
	dbq = dbq.Where("start_time > ? AND start_time < ?", query.StartTimeMin, query.StartTimeMax)
	if query.DurationMin != 0 {
		dbq = dbq.Where("duration > ?", query.DurationMin)
	}
	if query.DurationMax != 0 {
		dbq = dbq.Where("duration < ?", query.DurationMax)
	}
	err := dbq.Group("trace_id_low,trace_id_high").Limit(query.NumTraces).Select(&traceIds)
	if err != nil {
		return nil, err
	}
	traces := make([]*model.Trace, 0)
	for _, traceId := range traceIds {
		trace, _ := s.GetTrace(model.TraceID{Low: traceId.TraceIDLow, High: traceId.TraceIDHigh})
		traces = append(traces, trace)
	}
	return traces, nil
}
