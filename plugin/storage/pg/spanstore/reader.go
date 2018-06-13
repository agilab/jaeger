package spanstore

import (
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

/*
 * 作者:张晓明 时间:18/6/13
 */

type SpanReader struct {
}

func (SpanReader) GetTrace(traceID model.TraceID) (*model.Trace, error) {
	panic("implement me")
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
