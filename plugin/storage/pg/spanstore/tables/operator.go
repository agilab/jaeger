package tables

import "time"

/*
 * 作者:张晓明 时间:18/6/7
 */

type Operator struct {
	ID               int64
	OperatorID       int64
	OperatorName     string
	OperatorTypeID   int64
	OperatorTypeName string
	ServiceID        int64
	ServiceName      string
	CreatedAt        *time.Time
}
