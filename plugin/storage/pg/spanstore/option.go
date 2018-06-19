package spanstore

import (
	"time"
)

/*
 * 作者:张晓明 时间:18/6/12
 */

type WriterOption struct {
	RequestLogKey       string
	ResponseLogKey      string
	UserIdTagKey        string
	MaxBatchLen         int           //向数据库里面批量插入的最多数据条数
	BufferFlushInterval time.Duration //数据等待的最长时间
}
