package da

import (
	"github.com/jmoiron/sqlx"
	"log"
)

var Db *sqlx.DB

func init() {
	db, err := sqlx.Open("mysql", "root@tcp(127.0.0.1:3306)/sp_forum?charset=utf8")
	if err != nil {
		log.Fatalf("open mysql error:%v", err)
		return
	}
	Db = db
}
