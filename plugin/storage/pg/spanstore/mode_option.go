package spanstore

import (
	"github.com/jaegertracing/jaeger/model"
)

/*
 * 作者:张晓明 时间:18/6/12
 */

type GetStringFromSpanFnType func(span *model.Span) string
type GetInt64FromSpanFnType func(span *model.Span) int64

type modeOption struct {
	getRequestFn  GetStringFromSpanFnType
	getResponseFn GetStringFromSpanFnType
	getUserIDFn   GetInt64FromSpanFnType
}
