package pgtestutils

import (
	"github.com/go-pg/pg"
	"github.com/spf13/viper"
)

func NewDB() *pg.DB {
	viper.Set("db.url", "47.96.24.203:5432")
	viper.Set("db.user", "postgres")
	viper.Set("db.password", "sz1234")
	viper.Set("db.database", "postgres")
	db := pg.Connect(&pg.Options{
		Addr:     viper.GetString("db.url"),
		User:     viper.GetString("db.user"),
		Password: viper.GetString("db.password"),
		Database: viper.GetString("db.database"),
	})
	//db.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
	//	query, err := event.FormattedQuery()
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	log.Printf("%s %s", time.Since(event.StartTime), query)
	//})
	return db
}
