package da

import (
	es "github.com/olivere/elastic"
	"log"
)

var Es *es.Client

func init() {
	cli, err := es.NewClient(es.SetURL("localhost:9200"))
	if err != nil {
		log.Printf("connect to es error:%v", err)
		return
	}
	Es = cli
}
