package pgutil

import (
	"fmt"
	"time"

	"github.com/agilab/haunt_be/pkg/resource"
)

/*
 * 作者:张晓明 时间:18/6/8
 */

func CreatePartitionTableFromTo(startTime, endTime time.Time) {
	tbTime := startTime
	for {
		if tbTime.After(endTime) {
			break
		}
		CreatePartitionSpanTable(tbTime)
		tbTime = tbTime.Add(time.Hour)
	}
}

func CreatePartitionSpanTable(startTime time.Time) error {
	tablename := GetPartitionTableName("spans", startTime)
	sql := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS "public"."%s" PARTITION OF "public"."spans"
FOR VALUES FROM ('%s') TO ('%s')
;
ALTER TABLE "public"."%s" OWNER TO "postgres";
`, tablename, GetStartTimeFmt(startTime), GetEndTimeFmt(startTime), tablename)
	fmt.Println(tablename, GetStartTimeFmt(startTime), GetEndTimeFmt(startTime))
	_, err := resource.DB.Exec(sql)
	return err
}

func GetPartitionTableName(parentName string, startTime time.Time) string {
	return parentName + "_" + startTime.Format("2006010215")
}

func GetStartTimeFmt(startTime time.Time) string {
	trimTime := TrimTime2Houer(startTime)
	return trimTime.Format("2006-01-02 15:04:05")
}
func GetEndTimeFmt(startTime time.Time) string {
	trimTime := TrimTime2Houer(startTime)
	return trimTime.Add(time.Hour).Format("2006-01-02 15:04:05")
}

func TrimTime2Houer(startTime time.Time) time.Time {
	trimTime, _ := time.ParseInLocation("2006-01-02 15", startTime.Format("2006-01-02 15"), time.Local)
	return trimTime
}
