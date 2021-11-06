package service

import (
	"database/sql"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"sp-forum-back/da"
	"sp-forum-back/model"
	ec "sp-forum-back/model"
	"strconv"
)

type HomeService struct{}

func (hs *HomeService) Hots(c *gin.Context) ([]*model.Hot, *ec.E) {
	const hotNum = 6
	conn := da.OpenRedis()
	defer conn.Close()
	tid2score, err := redis.StringMap(conn.Do("zrevrange", "thread:rank", 0, hotNum-1, "WITHSCORES"))
	if err != nil {
		return nil, ec.RedisErr
	}
	hots := make([]*model.Hot, 0, hotNum)
	set := make(map[int]struct{})
	for tid, score := range tid2score {
		if s, err := strconv.ParseFloat(score, 64); err == nil && s >= 1 {
			t, _ := strconv.Atoi(tid)
			row := da.Db.QueryRow("select tid,title,`date` from sp_forum.thread where tid=?", t)
			var title, date string
			if err := row.Scan(&title, &date); err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return nil, ec.MysqlErr
			}
			hots = append(hots, &model.Hot{
				Tid:   t,
				Title: title,
				Date:  date,
			})
			set[t] = struct{}{}
		}
	}
	log.Printf("hots from score:%v", hots)
	tidsByLiked, err := redis.Ints(conn.Do("zrevrange", "thread:liked", 0, hotNum+5))
	if err != nil {
		return hots, nil
	}
	for _, tid := range tidsByLiked {
		if _, has := set[tid]; !has {
			row := da.Db.QueryRow("select tid,title,`date` from sp_forum.thread where tid=?", tid)
			var title, date string
			if err := row.Scan(&title, &date); err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return nil, ec.MysqlErr
			}
			hots = append(hots, &model.Hot{
				Tid:   tid,
				Title: title,
				Date:  date,
			})
			set[tid] = struct{}{}
		}
	}
	return hots, nil
}
