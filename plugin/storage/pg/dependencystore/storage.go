package dependencystore

import (
	"context"
	"time"

	gopg "github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/model"
	"go.uber.org/zap"
)

type DependencyStore struct {
	ctx    context.Context
	db     *gopg.DB
	logger *zap.Logger
}

// NewDependencyStore returns a DependencyStore
func NewDependencyStore(db *gopg.DB, logger *zap.Logger) *DependencyStore {
	return &DependencyStore{
		ctx:    context.Background(),
		db:     db,
		logger: logger,
	}
}

// GetDependencies returns all interservice dependencies
func (s *DependencyStore) GetDependencies(endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	// searchResult, err := s.client.Search(getIndices(endTs, lookback)...).
	// 	Type(dependencyType).
	// 	Size(10000). // the default elasticsearch allowed limit
	// 	Query(buildTSQuery(endTs, lookback)).
	// 	IgnoreUnavailable(true).
	// 	Do(s.ctx)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "Failed to search for dependencies")
	// }

	var retDependencies []model.DependencyLink
	// hits := searchResult.Hits.Hits
	// for _, hit := range hits {
	// 	source := hit.Source
	// 	var tToD timeToDependencies
	// 	if err := json.Unmarshal(*source, &tToD); err != nil {
	// 		return nil, errors.New("Unmarshalling ElasticSearch documents failed")
	// 	}
	// 	retDependencies = append(retDependencies, tToD.Dependencies...)
	// }
	return retDependencies, nil
}
