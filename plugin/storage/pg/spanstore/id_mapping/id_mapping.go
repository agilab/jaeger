package id_mapping

import (
	"fmt"

	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/tables"
)

/*
 * 作者:张晓明 时间:18/6/9
 */

type IDMappingService interface {
	GetIDFromName(serviceName string, metaType int64) int64
	GetNameFromID(id int64, metaType int64) string
}

var idMappingService IDMappingService

const CacheMaxSize = 1024 * 1024

func InitAndGetIDMappingService(db *pg.DB) IDMappingService {
	if idMappingService == nil {
		idMappingServiceImp := &GlobalIDMapping{}
		idMappingServiceImp.db = db
		idMappingServiceImp.lruCache = cache.NewLRU(CacheMaxSize)
		idMappingServiceImp.LoadData()
		idMappingService = idMappingServiceImp
	}
	return idMappingService
}

type GlobalIDMapping struct {
	lruCache cache.Cache
	db       *pg.DB
}

func makeNameKey(metaType int64, name string) string {
	return fmt.Sprintf("n_%d_%s", metaType, name)
}

func makeIdKey(metaType int64, id int64) string {
	return fmt.Sprintf("i_%d_%d", metaType, id)
}

func (g *GlobalIDMapping) LoadData() error {
	var opMetas []*tables.OpMeta
	err := g.db.Model(&opMetas).Limit(CacheMaxSize).Select()
	if err != nil {
		return err
	}
	for _, opMeta := range opMetas {
		g.lruCache.Put(makeNameKey(opMeta.Type, opMeta.Name), opMeta.Id)
		g.lruCache.Put(makeIdKey(opMeta.Type, opMeta.Id), opMeta.Name)
	}
	return nil
}

func (g *GlobalIDMapping) saveItem(name string, metaType int64) error {
	opMeta := &tables.OpMeta{Name: name, Type: metaType}
	_, err := g.db.Model(opMeta).Where("name = ?name and type = ?type").
		OnConflict("DO NOTHING").Returning("*").SelectOrInsert()
	if err != nil {
		return err
	}
	g.lruCache.Put(makeNameKey(opMeta.Type, opMeta.Name), opMeta.Id)
	g.lruCache.Put(makeIdKey(opMeta.Type, opMeta.Id), opMeta.Name)
	return nil
}

func (g GlobalIDMapping) GetIDFromName(name string, metaType int64) int64 {
	idObj := g.lruCache.Get(makeNameKey(metaType, name))
	id, ok := idObj.(int64)
	if ok {
		return id
	}
	g.saveItem(name, metaType)
	idObj = g.lruCache.Get(makeNameKey(metaType, name))
	id, ok = idObj.(int64)
	if ok {
		return id
	}
	return 0
}

func (g GlobalIDMapping) GetNameFromID(id int64, metaType int64) string {
	nameObj := g.lruCache.Get(makeIdKey(metaType, id))
	name, ok := nameObj.(string)
	if ok {
		return name
	}
	return "Unknown"
}
