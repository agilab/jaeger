package pg

type DB interface {
	// IndexExists(index string) IndicesExistsService
	// CreateIndex(index string) IndicesCreateService
	// Index() IndexService
	// Search(indices ...string) SearchService
	// MultiSearch() MultiSearchService
	// io.Closer
}
