package tables

import (
	"fmt"
	"testing"

	"github.com/go-pg/pg"
	"github.com/issue9/assert"
)

/*
 * 作者:张晓明 时间:18/6/7
 */

func Test_Span_table_create(t *testing.T) {
	db := pg.Connect(&pg.Options{
		Addr:     "47.96.24.203:5432",
		User:     "postgres",
		Password: "sz1234",
		Database: "postgres",
	})
	var err error
	err = db.CreateTable(&Operator{}, nil)
	err = db.CreateTable(&Span{}, nil)
	fmt.Println(err)
}

func Test_span_batch_insert(t *testing.T) {
	db := pg.Connect(&pg.Options{
		Addr:     "47.96.24.203:5432",
		User:     "postgres",
		Password: "sz1234",
		Database: "postgres",
	})
	spans := []Span{
		{
			Request: "testRequest1",
			TraceID: 123456,
			SpanID:  567890,
		},
		{
			Request: "testRequest1",
			TraceID: 1234563,
			SpanID:  5678904,
		},
	}
	err := db.Insert(&spans)
	assert.Nil(t, err)
}
