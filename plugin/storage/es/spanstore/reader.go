// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spanstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
	"gopkg.in/olivere/elastic.v5"

	"log"

	"github.com/jaegertracing/jaeger/model"
	jConverter "github.com/jaegertracing/jaeger/model/converter/json"
	jModel "github.com/jaegertracing/jaeger/model/json"
	"github.com/jaegertracing/jaeger/pkg/es"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	storageMetrics "github.com/jaegertracing/jaeger/storage/spanstore/metrics"
)

const (
	spanIndexPrefix    = "jaeger-span-"
	serviceIndexPrefix = "jaeger-service-"
	traceIDAggregation = "traceIDs"

	traceIDField       = "traceID"
	spanIDField        = "spanID"
	durationField      = "duration"
	startTimeField     = "startTime"
	serviceNameField   = "process.serviceName"
	operationNameField = "operationName"
	tagsField          = "tags"
	processTagsField   = "process.tags"
	logFieldsField     = "logs.fields"
	tagKeyField        = "key"
	tagValueField      = "value"
	requestMetaField   = "RequestMeta"
	requestBodyField   = "RequestBody"

	defaultDocCount  = 10000 // the default elasticsearch allowed limit
	defaultNumTraces = 100
)

var (
	// ErrServiceNameNotSet occurs when attempting to query with an empty service name
	ErrServiceNameNotSet = errors.New("Service Name must be set")

	// ErrStartTimeMinGreaterThanMax occurs when start time min is above start time max
	ErrStartTimeMinGreaterThanMax = errors.New("Start Time Minimum is above Maximum")

	// ErrDurationMinGreaterThanMax occurs when duration min is above duration max
	ErrDurationMinGreaterThanMax = errors.New("Duration Minimum is above Maximum")

	// ErrMalformedRequestObject occurs when a request object is nil
	ErrMalformedRequestObject = errors.New("Malformed request object")

	// ErrStartAndEndTimeNotSet occurs when start time and end time are not set
	ErrStartAndEndTimeNotSet = errors.New("Start and End Time must be set")

	// ErrUnableToFindTraceIDAggregation occurs when an aggregation query for TraceIDs fail.
	ErrUnableToFindTraceIDAggregation = errors.New("Could not find aggregation of traceIDs")

	errNoTraces = errors.New("No trace with that ID found")

	defaultMaxDuration = model.DurationAsMicroseconds(time.Hour * 24)

	tagFieldList = []string{tagsField, processTagsField, logFieldsField}
)

// SpanReader can query for and load traces from ElasticSearch
type SpanReader struct {
	ctx    context.Context
	client es.Client
	logger *zap.Logger
	// The age of the oldest service/operation we will look for. Because indices in ElasticSearch are by day,
	// this will be rounded down to UTC 00:00 of that day.
	maxLookback             time.Duration
	serviceOperationStorage *ServiceOperationStorage
}

// NewSpanReader returns a new SpanReader with a metrics.
func NewSpanReader(client es.Client, logger *zap.Logger, maxLookback time.Duration, metricsFactory metrics.Factory) spanstore.Reader {
	return storageMetrics.NewReadMetricsDecorator(newSpanReader(client, logger, maxLookback), metricsFactory)
}

func newSpanReader(client es.Client, logger *zap.Logger, maxLookback time.Duration) *SpanReader {
	ctx := context.Background()
	return &SpanReader{
		ctx:                     ctx,
		client:                  client,
		logger:                  logger,
		maxLookback:             maxLookback,
		serviceOperationStorage: NewServiceOperationStorage(ctx, client, metrics.NullFactory, logger, 0), // the decorator takes care of metrics
	}
}

// GetTrace takes a traceID and returns a Trace associated with that traceID
func (s *SpanReader) GetTrace(traceID model.TraceID) (*model.Trace, error) {
	currentTime := time.Now()
	traces, err := s.multiRead([]string{traceID.String()}, currentTime.Add(-s.maxLookback), currentTime, nil)
	if err != nil {
		return nil, err
	}
	if len(traces) == 0 {
		return nil, errNoTraces
	}
	return traces[0], nil
}

func (s *SpanReader) collectSpans(esSpansRaw []*elastic.SearchHit) ([]*model.Span, error) {
	spans := make([]*model.Span, len(esSpansRaw))

	for i, esSpanRaw := range esSpansRaw {
		jsonSpan, err := s.unmarshalJSONSpan(esSpanRaw)
		if err != nil {
			return nil, errors.Wrap(err, "Marshalling JSON to span object failed")
		}
		span, err := jConverter.SpanToDomain(jsonSpan)
		if err != nil {
			return nil, errors.Wrap(err, "Converting JSONSpan to domain Span failed")
		}
		spans[i] = span
	}
	return spans, nil
}

func (s *SpanReader) unmarshalJSONSpan(esSpanRaw *elastic.SearchHit) (*jModel.Span, error) {
	esSpanInByteArray := esSpanRaw.Source

	var jsonSpan jModel.Span
	if err := json.Unmarshal(*esSpanInByteArray, &jsonSpan); err != nil {
		return nil, err
	}
	return &jsonSpan, nil
}

// Returns the array of indices that we need to query, based on query params
func findIndices(prefix string, startTime time.Time, endTime time.Time) []string {
	var indices []string
	firstIndex := indexWithDate(prefix, startTime)
	currentIndex := indexWithDate(prefix, endTime)
	for currentIndex != firstIndex {
		indices = append(indices, currentIndex)
		endTime = endTime.Add(-24 * time.Hour)
		currentIndex = indexWithDate(prefix, endTime)
	}
	return append(indices, firstIndex)
}

func indexWithDate(prefix string, date time.Time) string {
	return prefix + date.UTC().Format("2006-01-02")
}

// GetServices returns all services traced by Jaeger, ordered by frequency
func (s *SpanReader) GetServices() ([]string, error) {
	currentTime := time.Now()
	jaegerIndices := findIndices(serviceIndexPrefix, currentTime.Add(-s.maxLookback), currentTime)
	return s.serviceOperationStorage.getServices(jaegerIndices)
}

// GetOperations returns all operations for a specific service traced by Jaeger
func (s *SpanReader) GetOperations(service string) ([]string, error) {
	currentTime := time.Now()
	jaegerIndices := findIndices(serviceIndexPrefix, currentTime.Add(-s.maxLookback), currentTime)
	return s.serviceOperationStorage.getOperations(jaegerIndices, service)
}

func bucketToStringArray(buckets []*elastic.AggregationBucketKeyItem) ([]string, error) {
	strings := make([]string, len(buckets))
	for i, keyitem := range buckets {
		str, ok := keyitem.Key.(string)
		if !ok {
			return nil, errors.New("Non-string key found in aggregation")
		}
		strings[i] = str
	}
	return strings, nil
}

func (s *SpanReader) GetSpans(findType, baseSpanID string, StartTimeMin, StartTimeMax time.Time) ([]*model.Span, error) {
	indices := findIndices(spanIndexPrefix, StartTimeMin.Add(-time.Hour), StartTimeMax.Add(time.Hour))
	query := elastic.NewTermQuery("spanID", baseSpanID)
	result, err := s.client.Search(indices...).Query(query).Do(s.ctx)
	if err != nil {
		return nil, err
	}
	if len(result.Hits.Hits) == 0 {
		return nil, errors.New("baseSpanID不存在")
	}
	fetchSpans, err := s.collectSpans(result.Hits.Hits)
	baseSpan := fetchSpans[0]
	var spanQuery elastic.Query
	switch findType {
	case "before":
		spanQuery = elastic.NewBoolQuery().Must(
			elastic.NewTermQuery("parentSpanID", baseSpan.ParentSpanID),
			elastic.NewRangeQuery("startTimeMillis").Lt(baseSpan.StartTime.Unix()*1000))
	case "after":
		spanQuery = elastic.NewBoolQuery().Must(
			elastic.NewTermQuery("parentSpanID", baseSpan.ParentSpanID),
			elastic.NewRangeQuery("startTimeMillis").Gt(baseSpan.StartTime.Unix()*1000))
	case "child":
		spanQuery = elastic.NewTermQuery("parentSpanID", baseSpan.SpanID)
	}
	spanResults, err := s.client.Search(indices...).Query(elastic.NewSearchSource().Query(spanQuery).
		Sort("startTime", true)).Do(s.ctx)
	if err != nil {
		return nil, err
	}
	if len(spanResults.Hits.Hits) == 0 {
		return []*model.Span{}, nil
	}
	fetchSpans, err = s.collectSpans(result.Hits.Hits)
	return fetchSpans, err
}

// FindTraces retrieves traces that match the traceQuery
func (s *SpanReader) FindTraces(traceQuery *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	if err := validateQuery(traceQuery); err != nil {
		return nil, err
	}
	if traceQuery.NumTraces == 0 {
		traceQuery.NumTraces = defaultNumTraces
	}
	uniqueTraceIDs, err := s.findTraceIDs(traceQuery)
	if err != nil {
		return nil, err
	}
	return s.multiRead(uniqueTraceIDs, traceQuery.StartTimeMin, traceQuery.StartTimeMax, traceQuery)
}

func (s *SpanReader) multiRead(traceIDs []string, startTime, endTime time.Time, traceQuery *spanstore.TraceQueryParameters) ([]*model.Trace, error) {

	if len(traceIDs) == 0 {
		return []*model.Trace{}, nil
	}
	searchRequests := make([]*elastic.SearchRequest, len(traceIDs))

	var traces []*model.Trace
	// Add an hour in both directions so that traces that straddle two indexes are retrieved.
	// i.e starts in one and ends in another.
	indices := findIndices(spanIndexPrefix, startTime.Add(-time.Hour), endTime.Add(time.Hour))

	nextTime := model.TimeAsEpochMicroseconds(startTime.Add(-time.Hour))

	searchAfterTime := make(map[string]uint64)
	totalDocumentsFetched := make(map[string]int)
	tracesMap := make(map[string]*model.Trace)

	var parentSpanIDMap = make(map[model.SpanID]bool)
	for {
		if traceIDs == nil || len(traceIDs) == 0 {
			break
		}

		for i, traceID := range traceIDs {
			var query elastic.Query
			query = elastic.NewTermQuery("traceID", traceID)
			if traceQuery != nil && traceQuery.Lazy != 0 {
				mustQuery := elastic.NewBoolQuery().Must(elastic.NewTermQuery("traceID", traceID))
				mustQuery.Must(s.buildFindTraceIDsQuery(traceQuery))
				query = mustQuery
			}
			if val, ok := searchAfterTime[traceID]; ok {
				nextTime = val
			}
			searchRequests[i] = elastic.NewSearchRequest().IgnoreUnavailable(true).Type("span").Source(elastic.NewSearchSource().Query(query).Size(defaultDocCount).Sort("startTime", true).SearchAfter(nextTime))
		}
		// set traceIDs to empty
		traceIDs = nil
		results, err := s.client.MultiSearch().Add(searchRequests...).Index(indices...).Do(s.ctx)

		if err != nil {
			return nil, err
		}

		if results.Responses == nil || len(results.Responses) == 0 {
			break
		}

		for _, result := range results.Responses {
			if result.Hits == nil || len(result.Hits.Hits) == 0 {
				continue
			}
			spans, err := s.collectSpans(result.Hits.Hits)
			if err != nil {
				return nil, err
			}
			lastSpan := spans[len(spans)-1]
			lastSpanTraceID := lastSpan.TraceID.String()

			if traceSpan, ok := tracesMap[lastSpanTraceID]; ok {
				for _, span := range spans {
					if _, ok := parentSpanIDMap[span.ParentSpanID]; !ok {
						if span.SpanID != span.ParentSpanID && span.ParentSpanID != 0 {
							parentSpanIDMap[span.ParentSpanID] = true
						}
					}
					traceSpan.Spans = append(traceSpan.Spans, span)
				}

			} else {
				for _, span := range spans {
					if _, ok := parentSpanIDMap[span.ParentSpanID]; !ok {
						if span.SpanID != span.ParentSpanID && span.ParentSpanID != 0 {
							parentSpanIDMap[span.ParentSpanID] = true
						}
					}
				}
				tracesMap[lastSpanTraceID] = &model.Trace{Spans: spans}
			}

			totalDocumentsFetched[lastSpanTraceID] = totalDocumentsFetched[lastSpanTraceID] + len(result.Hits.Hits)
			if totalDocumentsFetched[lastSpanTraceID] < int(result.TotalHits()) {
				traceIDs = append(traceIDs, lastSpanTraceID)
				searchAfterTime[lastSpanTraceID] = model.TimeAsEpochMicroseconds(lastSpan.StartTime)
			}
		}
	}
	for {
		var parentSpanIDs []interface{}
		for parentSpanID := range parentSpanIDMap {
			parentSpanIDs = append(parentSpanIDs, parentSpanID)
			log.Println(parentSpanIDs)
		}
		//TODO 目前只支持1w个以内的懒加载模式
		searchResults, err := s.client.Search(indices...).Query(elastic.NewTermsQuery("spanID", parentSpanIDs...)).Size(defaultDocCount).Do(s.ctx)
		if err != nil {
			return nil, err
		}
		if searchResults == nil || len(searchResults.Hits.Hits) == 0 {
			break
		}
		spans, err := s.collectSpans(searchResults.Hits.Hits)
		if err != nil {
			return nil, err
		}
		parentSpanIDMap = make(map[model.SpanID]bool)
		for _, span := range spans {
			if _, ok := parentSpanIDMap[span.ParentSpanID]; !ok {
				if span.SpanID != span.ParentSpanID && span.ParentSpanID != 0 {
					parentSpanIDMap[span.ParentSpanID] = true
				}
			}
			tracesMap[span.TraceID.String()].Spans = append(tracesMap[span.TraceID.String()].Spans, span)
		}
		if len(parentSpanIDMap) == 0 {
			break
		}
	}

	for _, trace := range tracesMap {
		traces = append(traces, trace)
	}
	return traces, nil
}

func validateQuery(p *spanstore.TraceQueryParameters) error {
	if p == nil {
		return ErrMalformedRequestObject
	}
	if p.ServiceName == "" && len(p.Tags) > 0 && p.OperationName == "" {
		return ErrServiceNameNotSet
	}
	if p.StartTimeMin.IsZero() || p.StartTimeMax.IsZero() {
		return ErrStartAndEndTimeNotSet
	}
	if p.StartTimeMax.Before(p.StartTimeMin) {
		return ErrStartTimeMinGreaterThanMax
	}
	if p.DurationMin != 0 && p.DurationMax != 0 && p.DurationMin > p.DurationMax {
		return ErrDurationMinGreaterThanMax
	}
	return nil
}

func (s *SpanReader) findTraceIDs(traceQuery *spanstore.TraceQueryParameters) ([]string, error) {
	//  Below is the JSON body to our HTTP GET request to ElasticSearch. This function creates this.
	// {
	//      "size": 0,
	//      "query": {
	//        "bool": {
	//          "must": [
	//            { "match": { "operationName":   "op1"      }},
	//            { "match": { "process.serviceName": "service1" }},
	//            { "range":  { "startTime": { "gte": 0, "lte": 90000000000000000 }}},
	//            { "range":  { "duration": { "gte": 0, "lte": 90000000000000000 }}},
	//            { "should": [
	//                   { "nested" : {
	//                      "path" : "tags",
	//                      "query" : {
	//                          "bool" : {
	//                              "must" : [
	//                              { "match" : {"tags.key" : "tag3"} },
	//                              { "match" : {"tags.value" : "xyz"} }
	//                              ]
	//                          }}}},
	//                   { "nested" : {
	//                          "path" : "process.tags",
	//                          "query" : {
	//                              "bool" : {
	//                                  "must" : [
	//                                  { "match" : {"tags.key" : "tag3"} },
	//                                  { "match" : {"tags.value" : "xyz"} }
	//                                  ]
	//                              }}}},
	//                   { "nested" : {
	//                          "path" : "logs.fields",
	//                          "query" : {
	//                              "bool" : {
	//                                  "must" : [
	//                                  { "match" : {"tags.key" : "tag3"} },
	//                                  { "match" : {"tags.value" : "xyz"} }
	//                                  ]
	//                              }}}}
	//                ]
	//              }
	//          ]
	//        }
	//      },
	//      "aggs": { "traceIDs" : { "terms" : {"size": 100,"field": "traceID" }}}
	//  }
	aggregation := s.buildTraceIDAggregation(traceQuery.NumTraces)
	boolQuery := s.buildFindTraceIDsQuery(traceQuery)

	jaegerIndices := findIndices(spanIndexPrefix, traceQuery.StartTimeMin, traceQuery.StartTimeMax)

	searchService := s.client.Search(jaegerIndices...).
		Type(spanType).
		Size(0). // set to 0 because we don't want actual documents.
		Aggregation(traceIDAggregation, aggregation).
		IgnoreUnavailable(true).
		Query(boolQuery)

	searchResult, err := searchService.Do(s.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Search service failed")
	}
	if searchResult.Aggregations == nil {
		return []string{}, nil
	}
	bucket, found := searchResult.Aggregations.Terms(traceIDAggregation)
	if !found {
		return nil, ErrUnableToFindTraceIDAggregation
	}

	traceIDBuckets := bucket.Buckets
	return bucketToStringArray(traceIDBuckets)
}

func (s *SpanReader) buildTraceIDAggregation(numOfTraces int) elastic.Aggregation {
	return elastic.NewTermsAggregation().
		Size(numOfTraces).
		Field(traceIDField).
		Order(startTimeField, false).
		SubAggregation(startTimeField, s.buildTraceIDSubAggregation())
}

func (s *SpanReader) buildTraceIDSubAggregation() elastic.Aggregation {
	return elastic.NewMaxAggregation().
		Field(startTimeField)
}

func (s *SpanReader) buildFindTraceIDsQuery(traceQuery *spanstore.TraceQueryParameters) elastic.Query {
	boolQuery := elastic.NewBoolQuery()

	//add duration query
	if traceQuery.DurationMax != 0 || traceQuery.DurationMin != 0 {
		durationQuery := s.buildDurationQuery(traceQuery.DurationMin, traceQuery.DurationMax)
		boolQuery.Must(durationQuery)
	}

	//add startTime query
	startTimeQuery := s.buildStartTimeQuery(traceQuery.StartTimeMin, traceQuery.StartTimeMax)
	boolQuery.Must(startTimeQuery)

	//add process.serviceName query
	if traceQuery.ServiceName != "" {
		serviceNameQuery := s.buildServiceNameQuery(traceQuery.ServiceName)
		boolQuery.Must(serviceNameQuery)
	}

	//add operationName query
	if traceQuery.OperationName != "" {
		operationNameQuery := s.buildOperationNameQuery(traceQuery.OperationName)
		boolQuery.Must(operationNameQuery)
	}
	if traceQuery.RequestMeta != "" {
		requestMetaQuery := s.buildRequestMetaQuery(traceQuery.RequestMeta)
		boolQuery.Must(requestMetaQuery)
	}

	if traceQuery.RequestMeta != "" {
		requestBodyQuery := s.buildRequestBodyQuery(traceQuery.RequestBody)
		boolQuery.Must(requestBodyQuery)
	}

	for k, v := range traceQuery.Tags {
		tagQuery := s.buildTagQuery(k, v)
		boolQuery.Must(tagQuery)
	}
	return boolQuery
}

func (s *SpanReader) buildDurationQuery(durationMin time.Duration, durationMax time.Duration) elastic.Query {
	minDurationMicros := model.DurationAsMicroseconds(durationMin)
	maxDurationMicros := defaultMaxDuration
	if durationMax != 0 {
		maxDurationMicros = model.DurationAsMicroseconds(durationMax)
	}
	return elastic.NewRangeQuery(durationField).Gte(minDurationMicros).Lte(maxDurationMicros)
}

func (s *SpanReader) buildStartTimeQuery(startTimeMin time.Time, startTimeMax time.Time) elastic.Query {
	minStartTimeMicros := model.TimeAsEpochMicroseconds(startTimeMin)
	maxStartTimeMicros := model.TimeAsEpochMicroseconds(startTimeMax)
	return elastic.NewRangeQuery(startTimeField).Gte(minStartTimeMicros).Lte(maxStartTimeMicros)
}

func (s *SpanReader) buildServiceNameQuery(serviceName string) elastic.Query {
	return elastic.NewMatchQuery(serviceNameField, serviceName)
}

func (s *SpanReader) buildRequestMetaQuery(requestMeta string) elastic.Query {
	return elastic.NewMatchQuery(requestMetaField, requestMeta)
}
func (s *SpanReader) buildRequestBodyQuery(requestBody string) elastic.Query {
	return elastic.NewMatchQuery(requestBodyField, requestBody)
}

func (s *SpanReader) buildOperationNameQuery(operationName string) elastic.Query {
	return elastic.NewMatchQuery(operationNameField, operationName)
}

func (s *SpanReader) buildTagQuery(k string, v string) elastic.Query {
	queries := make([]elastic.Query, len(tagFieldList))
	for i := range queries {
		queries[i] = s.buildNestedQuery(tagFieldList[i], k, v)
	}
	return elastic.NewBoolQuery().Should(queries...)
}

func (s *SpanReader) buildNestedQuery(field string, k string, v string) elastic.Query {
	keyField := fmt.Sprintf("%s.%s", field, tagKeyField)
	valueField := fmt.Sprintf("%s.%s", field, tagValueField)
	keyQuery := elastic.NewMatchQuery(keyField, k)
	valueQuery := elastic.NewMatchQuery(valueField, v)
	tagBoolQuery := elastic.NewBoolQuery().Must(keyQuery, valueQuery)
	return elastic.NewNestedQuery(field, tagBoolQuery)
}
