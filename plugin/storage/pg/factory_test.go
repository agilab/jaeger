package pg

import (
	"testing"

	"github.com/go-pg/pg"
	"github.com/spf13/viper"
)

func TestDBConnect(t *testing.T) {
	c := NewDB()
	if c == nil {
		panic("db connect error")
	}
}

func NewDB() *pg.DB {
	viper.Set("db.url", "localhost:5432")
	viper.Set("db.user", "postgres")
	viper.Set("db.password", "123456")
	viper.Set("db.database", "postgres")
	db := pg.Connect(&pg.Options{
		Addr:     viper.GetString("db.url"),
		User:     viper.GetString("db.user"),
		Password: viper.GetString("db.password"),
		Database: viper.GetString("db.database"),
	})
	return db
}
