package id_mapping

import (
	"testing"

	"github.com/agilab/haunt_be/pkg/resource"
	"github.com/agilab/haunt_be/pkg/tables"
	"github.com/issue9/assert"
)

/*
 * 作者:张晓明 时间:18/6/9
 */

func Test_GetMetaIDByName_servicename(t *testing.T) {
	resource.Init_resource()
	id1 := GetMetaIDByName("test_test_be", tables.OpMetaTypeEnum.Service)
	id2 := GetMetaIDByName("test_test_be", tables.OpMetaTypeEnum.Service)
	assert.Equal(t, id1, id2)
	assert.NotEqual(t, id1, 0)
}

func Test_GetMetaIDByName_optype(t *testing.T) {
	resource.Init_resource()
	id1 := GetMetaIDByName("optype", tables.OpMetaTypeEnum.OpType)
	id2 := GetMetaIDByName("optype", tables.OpMetaTypeEnum.OpType)
	assert.Equal(t, id1, id2)
	assert.NotEqual(t, id1, 0)
}

func Test_GetMetaIDByName_OpName(t *testing.T) {
	resource.Init_resource()
	id1 := GetMetaIDByName("optype", tables.OpMetaTypeEnum.OpName)
	id2 := GetMetaIDByName("optype", tables.OpMetaTypeEnum.OpName)
	assert.Equal(t, id1, id2)
	assert.NotEqual(t, id1, 0)
}

func Test_GetOperatorFromName(t *testing.T) {
	resource.Init_resource()
	op := InitAndGetIDMappingService().GetOperatorFromName("Test_GetOperatorFromName_service",
		"Test_GetOperatorFromName_typename", "Test_GetOperatorFromName_opname")
	assert.Equal(t, op.ServiceName, "Test_GetOperatorFromName_service")
	assert.Equal(t, op.OperatorTypeName, "Test_GetOperatorFromName_typename")
	assert.Equal(t, op.OperatorName, "Test_GetOperatorFromName_opname")
}
