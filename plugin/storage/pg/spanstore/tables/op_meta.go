package tables

/*
 * 作者:张晓明 时间:18/6/11
 */

type OpMeta struct {
	Name string
	Id   int64
	Type int64 //1-service,2-type,3-opName
}

var OpMetaTypeEnum = struct {
	Service int64
	OpType  int64
	OpName  int64
}{
	Service: 1,
	OpType:  2,
	OpName:  3,
}