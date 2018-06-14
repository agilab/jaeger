package spanstore

import (
	"testing"

	"github.com/go-pg/pg"
	"github.com/spf13/viper"
)

/*
 * 作者:张晓明 时间:18/6/13
 */

func TestSpanWriter_WriteSpan(t *testing.T) {
	viper.Set("db.url", "47.96.24.203:5432")
	viper.Set("user", "postgres")
	viper.Set("password", "sz1234")
	viper.Set("database", "postgres")
	_ = pg.Connect(&pg.Options{
		Addr:     viper.GetString("db.url"),
		User:     viper.GetString("db.user"),
		Password: viper.GetString("db.password"),
		Database: viper.GetString("db.database"),
	})
}
