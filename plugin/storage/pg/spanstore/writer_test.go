package spanstore

import (
	"encoding/json"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/testutils"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/id_mapping"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/testutils"
	"github.com/uber/jaeger-lib/metrics"
)

/*
 * 作者:张晓明 时间:18/6/13
 */

func Benchmark_SpanWriter_WriteSpan(b *testing.B) {
	db := pgtestutils.NewDB()
	logger, _ := testutils.NewLogger()
	id_mapping.InitAndGetIDMappingService(db)
	spanWriter := NewSpanWriter(db, logger, metrics.NullFactory, writerOption{
		requestLogKey:       "request",
		responseLogKey:      "response",
		userIdTagKey:        "user.id",
		maxBatchLen:         5000,
		bufferFlushInterval: time.Second,
	})
	logs := make([]model.Log, 0)
	log.Println(time.Now().Format("2006-01-02T15:04:05Z07:00"))
	json.Unmarshal([]byte(`[{
		"timestamp": "2018-06-18T23:00:23+08:00",
		"fields": [{
			"key": "metadata",
			"type": "string",
			"value": ""
		}]
	},
	{
		"timestamp": "2018-06-18T23:00:23+08:00",
		"fields": [{
			"key": "request.body",
			"type": "string",
			"value": ""
		}]
	},
	{
		"timestamp": "2018-06-18T23:00:23+08:00",
		"fields": [{
			"key": "response.body",
			"type": "string",
			"value": ""
		}]
	}
]`), &logs)
	mSpan := &model.Span{
		Process:       model.NewProcess("tesla_keyword_be", nil),
		TraceID:       model.TraceID{Low: 0x49e0e6d56552289b, High: 0},
		ParentSpanID:  model.SpanID(0x49e0e6d56552289b),
		Flags:         1,
		OperationName: "SQL SELECT base_keywords",
		References:    []model.SpanRef{},
		StartTime:     time.Now(),
		Duration:      time.Second,
		Tags: buildMap2Tags(map[string]interface{}{
			"component": "sql",
			"db.type":   "mysql",
			"db.method": "select",
			"db.table":  "base_keywords",
			"user.id":   "1005096613967839232",
		}),
		Warnings: []string{"warnings1"},
	}
	//start := time.Now()
	wg := sync.WaitGroup{}
	b.N = 20 * 10000
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < b.N/20; j++ {
				mSpan.SpanID = model.SpanID(i*1000000 + j)
				writeSpan(spanWriter, mSpan)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	spanWriter.Flush()
	//log.Println(time.Now().Sub(start))
}

func writeSpan(writer *SpanWriter, span *model.Span) {
	writer.WriteSpan(span)
}
