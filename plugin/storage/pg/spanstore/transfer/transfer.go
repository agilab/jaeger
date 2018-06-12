package transfer

import (
	"github.com/agilab/haunt_be/pkg/tables"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
)

/*
 * 作者:张晓明 时间:18/6/8
 */

func TransJSpan2TableSpan(span *jaeger.Span) *tables.Span {
	tSpan := new(tables.Span)
	tSpan.SpanID = span.SpanId
	tSpan.TraceID = span.TraceIdLow
	tSpan.Request = span.
	tSpan.UserID = span.UserID
	tSpan.ParentSpanIds = span.ParentSpanIDs
	tSpan.StartTime = span.StartTime
	tSpan.Duration = span.Duration
	tSpan.Tags = span.Tags
	tSpan.Logs = span.Logs
	tSpan.Flags = span.Flags
	tSpan.Warnings = span.Warnings
	tSpan.Response = span.Response
	op := id_mapping.InitAndGetIDMappingService().GetOperatorFromName(span.ServiceName, span.OperatorTypeName, span.OperatorName)
	tSpan.ServiceID = op.ServiceID
	tSpan.OperatorTypeID = op.OperatorTypeID
	tSpan.OperatorTypeID = op.OperatorTypeID
	tSpan.ParentOperatorIds = span.ParentOperatorIDs
	tSpan.Process = span.Process
	tSpan.Reference = span.Reference
	return tSpan
}

