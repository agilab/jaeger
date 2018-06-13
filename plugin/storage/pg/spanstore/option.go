package spanstore

import (
	"time"

	"github.com/jaegertracing/jaeger/model"
)

/*
 * 作者:张晓明 时间:18/6/12
 */

type GetStringFromSpanFnType func(span *model.Span) string
type GetInt64FromSpanFnType func(span *model.Span) int64

type writerOption struct {
	getRequestFn        GetStringFromSpanFnType
	getResponseFn       GetStringFromSpanFnType
	getUserIDFn         GetInt64FromSpanFnType
	maxBatchLen         int           //向数据库里面批量插入的最多数据条数
	bufferFlushInterval time.Duration //数据等待的最长时间
}
