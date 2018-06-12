package tables

import (
	"time"
	"github.com/jaegertracing/jaeger/model"
)

/*
 * 作者:张晓明 时间:18/6/7
 */

type Span struct {
	SpanID            int64
	TraceID           int64
	ParentSpanIds     []int64 `pg:",array"`
	Flags             int64
	OperatorID        int64
	OperatorTypeID    int64
	ServiceID         int64
	ParentOperatorIds []int64 `pg:",array"`
	StartTime         *time.Time
	Duration          int64
	Tags              map[string]interface{}
	Logs              []model.Log
	Process           map[string]string
	Reference         map[string]string
	Warnings          []string `pg:",array"`
	UserID            int64
	Response          string
	Request           string
	ErrorCode         int64
}
