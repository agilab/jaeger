package spanstore

import (
	"time"
)

/*
 * 作者:张晓明 时间:18/6/12
 */

type writerOption struct {
	requestLogKey       string
	responseLogKey      string
	userIdTagKey        string
	maxBatchLen         int           //向数据库里面批量插入的最多数据条数
	bufferFlushInterval time.Duration //数据等待的最长时间
}
