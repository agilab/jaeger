package id_mapping

import (
	"sync"

	"strconv"
	"strings"

	"log"

	"github.com/agilab/haunt_be/pkg/resource"
	"github.com/agilab/haunt_be/pkg/tables"
)

/*
 * 作者:张晓明 时间:18/6/9
 */

type IDMappingService interface {
	GetOperatorFromID(svcID, typeID, opID int64) *tables.Operator
	GetOperatorFromName(svcName, typeName, opName string) *tables.Operator
}

var idMappingService IDMappingService

func InitAndGetIDMappingService() IDMappingService {
	if idMappingService == nil {
		idMappingServiceImp := &GlobalIDMapping{}
		idMappingServiceImp.OperatorIDMappingMap = make(map[string]*tables.Operator)
		idMappingServiceImp.OperatorNameMappingMap = make(map[string]*tables.Operator)
		idMappingServiceImp.LoadData()
		idMappingService = idMappingServiceImp
	}
	return idMappingService
}

type GlobalIDMapping struct {
	// key值是三个ID，用 `-` 链接
	OperatorIDMappingMap map[string]*tables.Operator
	// key 是三个name,用 `-` 链接
	OperatorNameMappingMap map[string]*tables.Operator
	lock                   sync.Locker
}

func CombineName(op *tables.Operator) string {
	return combineName(op.ServiceName, op.OperatorTypeName, op.OperatorName)
}

func combineName(svcName, typeName, opName string) string {
	return strings.Join([]string{svcName, typeName, opName}, "-")
}
func CombineID(op *tables.Operator) string {
	return combineID(op.ServiceID, op.OperatorTypeID, op.OperatorID)
}
func combineID(svcID, typeID, opID int64) string {
	return strings.Join([]string{strconv.FormatInt(svcID, 10),
		strconv.FormatInt(typeID, 10),
		strconv.FormatInt(opID, 10)}, "-")
}

func (g *GlobalIDMapping) LoadData() error {
	g.lock = &sync.Mutex{}
	g.lock.Lock()
	defer g.lock.Unlock()
	var operators []*tables.Operator
	err := resource.DB.Model(&operators).Select()
	if err != nil {
		return err
	}
	for _, operator := range operators {
		g.OperatorIDMappingMap[CombineID(operator)] = operator
		g.OperatorNameMappingMap[CombineName(operator)] = operator
	}
	return nil
}

func (g *GlobalIDMapping) addOperator2Map(operator *tables.Operator) {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.OperatorIDMappingMap[CombineID(operator)] = operator
	g.OperatorNameMappingMap[CombineName(operator)] = operator
}

// 这个是用于保证每个类型的名字唯一
func GetMetaIDByName(metaName string, metaType int64) int64 {
	opMeta := tables.OpMeta{
		Name: metaName,
		Type: metaType,
	}
	_, err := resource.DB.Model(&opMeta).Where("name = ?name and type = ?type").
		OnConflict("DO NOTHING").Returning("*").SelectOrInsert()
	if err != nil {
		log.Printf("SelectOrInsert op_meta error:%s", err.Error())
	}
	return opMeta.Id
}

func (g GlobalIDMapping) GetOperatorFromName(svcName, typeName, opName string) *tables.Operator {
	nameKey := combineName(svcName, typeName, opName)
	op, ok := g.OperatorNameMappingMap[nameKey]
	if !ok {
		op = &tables.Operator{ServiceName: svcName, OperatorTypeName: typeName, OperatorName: opName}
		op.ServiceID = GetMetaIDByName(svcName, tables.OpMetaTypeEnum.Service)
		op.OperatorTypeID = GetMetaIDByName(typeName, tables.OpMetaTypeEnum.OpType)
		op.OperatorID = GetMetaIDByName(opName, tables.OpMetaTypeEnum.OpName)
		_, err := resource.DB.Model(op).Where("service_id = ?service_id and operator_type_id = ?operator_type_id and operator_id = ?operator_id").
			OnConflict("DO NOTHING").Returning("*").SelectOrInsert()
		if err != nil {
			log.Printf("SelectOrInsert operators error:%s", err.Error())
		}
		if op.ID != 0 {
			g.addOperator2Map(op)
		}
		return op
	}
	return op
}

func (g GlobalIDMapping) GetOperatorFromID(svcID, typeID, opID int64) *tables.Operator {
	idKey := combineID(svcID, typeID, opID)
	op, ok := g.OperatorIDMappingMap[idKey]
	if !ok {
		op = &tables.Operator{ServiceID: svcID, OperatorTypeID: typeID, OperatorID: opID}
		resource.DB.Model(&op).Where("service_id = ?service_id and operator_type_id = ?operator_type_id and operator_id = ?operator_id").
			Select()
		if op.ID != 0 {
			g.addOperator2Map(op)
		}
		return op
	}
	return op
}
