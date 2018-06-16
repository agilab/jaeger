package id_mapping

import (
	"fmt"

	"strconv"

	"log"

	"github.com/go-pg/pg"
	"github.com/jaegertracing/jaeger/pkg/cache"
	"github.com/jaegertracing/jaeger/plugin/storage/pg/spanstore/tables"
)

/*
 * 作者:张晓明 时间:18/6/9
 */

type IDMappingService interface {
	GetIdFromName(serviceName string, metaType int64) int64
	GetNameFromId(id int64, metaType int64) string
	RegisterIDRelation(serviceId, opTypeId, opNameId int64)
}

var idMappingService IDMappingService

const CacheMaxSize = 1024 * 1024

func InitAndGetIDMappingService(db *pg.DB) IDMappingService {
	if idMappingService == nil {
		idMappingServiceImp := &GlobalIDMapping{}
		idMappingServiceImp.db = db
		idMappingServiceImp.lruOpMetaCache = cache.NewLRU(CacheMaxSize)
		idMappingServiceImp.lruOpRelationCache = cache.NewLRU(CacheMaxSize)
		idMappingServiceImp.LoadData()
		idMappingService = idMappingServiceImp
	}
	return idMappingService
}

type GlobalIDMapping struct {
	lruOpMetaCache     cache.Cache
	lruOpRelationCache cache.Cache
	db                 *pg.DB
}

func (g *GlobalIDMapping) RegisterIDRelation(serviceId, opTypeId, opNameId int64) {
	cacheKey := strconv.FormatInt(serviceId, 36) + strconv.FormatInt(opTypeId, 36) + strconv.FormatInt(opNameId, 36)
	v := g.lruOpRelationCache.Get(cacheKey)
	if v != nil {
		return
	} else {
		g.lruOpRelationCache.Put(cacheKey, true)
		opRelation := &tables.OpRelation{
			ServiceId: serviceId,
			OpTypeId:  opTypeId,
			OpNameId:  opNameId,
		}
		_, err := g.db.Model(opRelation).OnConflict("DO NOTHING").Insert()
		if err != nil {
			log.Println("opRelation Insert Failed!", err)
		}
	}
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
		g.lruOpMetaCache.Put(makeNameKey(opMeta.Type, opMeta.Name), opMeta.Id)
		g.lruOpMetaCache.Put(makeIdKey(opMeta.Type, opMeta.Id), opMeta.Name)
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
	g.lruOpMetaCache.Put(makeNameKey(opMeta.Type, opMeta.Name), opMeta.Id)
	g.lruOpMetaCache.Put(makeIdKey(opMeta.Type, opMeta.Id), opMeta.Name)
	return nil
}

func (g GlobalIDMapping) GetIdFromName(name string, metaType int64) int64 {
	idObj := g.lruOpMetaCache.Get(makeNameKey(metaType, name))
	id, ok := idObj.(int64)
	if ok {
		return id
	}
	g.saveItem(name, metaType)
	idObj = g.lruOpMetaCache.Get(makeNameKey(metaType, name))
	id, ok = idObj.(int64)
	if ok {
		return id
	}
	return 0
}

func (g GlobalIDMapping) GetNameFromId(id int64, metaType int64) string {
	nameObj := g.lruOpMetaCache.Get(makeIdKey(metaType, id))
	name, ok := nameObj.(string)
	if ok {
		return name
	}
	return "Unknown"
}
