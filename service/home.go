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

func (hs *HomeService) Hots(_ *gin.Context) ([]*model.Hot, *ec.E) {
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
			row := da.Db.QueryRow("select title,`date` from sp_forum.thread where tid=?", t)
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
	tidsByLiked, err := redis.Ints(conn.Do("zrevrange", "thread:liked", 0, hotNum+5))
	if err != nil {
		return hots, nil
	}
	for _, tid := range tidsByLiked {
		if _, has := set[tid]; !has {
			row := da.Db.QueryRow("select title,`date` from sp_forum.thread where tid=?", tid)
			var title, date string
			if err := row.Scan(&title, &date); err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				log.Printf("exec mysql error:%v", err)
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

func (hs *HomeService) Carousel(_ *gin.Context) ([]*model.Carousel, *ec.E) {
	rows, err := da.Db.Query("select tid,`desc`,picture_url from sp_forum.carousel")
	if err != nil {
		return nil, ec.MysqlErr
	}
	carousels := make([]*model.Carousel, 0)
	for rows.Next() {
		c := &model.Carousel{}
		if err := rows.Scan(&c.Tid, &c.Title, &c.PictureUrl); err != nil {
			return nil, ec.MysqlErr
		}
		carousels = append(carousels, c)
	}
	return carousels, nil
}

func (hs *HomeService) Posts(c *gin.Context, page int) ([]*model.Post, *ec.E) {
	const size = 10
	rows, err := da.Db.Query("select a.tid,title,last_modify,u.uid,u.username,u.face_url from thread a join(select tid from thread order by last_modify desc limit ?,?)b on a.tid=b.tid join user u on u.uid=a.author",
		(page-1)*size, size)
	if err != nil {
		log.Printf("exec mysql error%v", err)
		return nil, ec.MysqlErr
	}
	posts := make([]*model.Post, 0)
	for rows.Next() {
		p := &model.Post{
			Author: model.Author{},
		}
		if err := rows.Scan(&p.Tid, &p.Title, &p.LastModify, &p.Author.Uid, &p.Author.Username, &p.Author.FaceUrl); err != nil {
			log.Printf("scan mysql error:%v", err)
			continue
		}
		if e := PostDetail(p); e == nil {
			posts = append(posts, p)
		}
	}
	return posts, nil
}

func PostDetail(p *model.Post) *ec.E {
	row := da.Db.QueryRow("select content from level where thread=? and is_root=1", p.Tid)
	if err := row.Scan(&p.Abstract); err != nil {
		log.Printf("scan mysql error:%v", err)
		return ec.MysqlErr
	}
	row = da.Db.QueryRow("select sum(reply_num) from level where thread=?", p.Tid)
	if err := row.Scan(&p.ReplyNum); err != nil {
		log.Printf("scan mysql error:%v", err)
		return ec.MysqlErr
	}
	row = da.Db.QueryRow("select count(uid) from visit where tid=?", p.Tid)
	if err := row.Scan(&p.VisitNum); err != nil {
		log.Printf("scan mysql error:%v", err)
		return ec.MysqlErr
	}
	return nil
}
