package service

import (
	"database/sql"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"math"
	"sp-forum-back/da"
	"sp-forum-back/model"
	ec "sp-forum-back/model"
	"strconv"
	"time"
)

type threadService interface {
	GetThread(c *gin.Context, tid int) (*model.Thread, *ec.E)
	GetReply(c *gin.Context, lid int) ([]*model.Reply, *ec.E)
	CreateLevel(c *gin.Context, tid int, content string) (*model.Level, *ec.E)
	CreateThread(c *gin.Context, title string) (*model.Thread, *ec.E)
	CreateReply(c *gin.Context, lid int, content string, to int) (*model.Reply, *ec.E)
	DelReply(c *gin.Context, rid int) *ec.E
	DelLevel(c *gin.Context, lid int) *ec.E
	DelThread(c *gin.Context, tid int) *ec.E
	Fav(c *gin.Context, lid, rid int) *ec.E
	Collect(c *gin.Context, tid int) *ec.E
}

type ThreadService struct{}

func (ts *ThreadService) GetThread(_ *gin.Context, tid int) (*model.Thread, *ec.E) {
	t := &model.Thread{}

	sqlStr := "select tid,title,`date`,fav,reply_num,last_modify,u.uid,u.username,u.face_url" +
		" from thread t join user u on t.author=u.uid where tid=?"
	row := da.Db.QueryRow(sqlStr, tid)
	err := row.Scan(&t.Tid, &t.Title, &t.Date, &t.Fav, &t.ReplyNum, &t.LastModify, &t.Author.Uid, &t.Author.Username, &t.Author.FaceUrl)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ec.ThreadNotExist
		}
		return nil, ec.MysqlErr
	}
	levels := make([]model.Level, 0)
	sqlStr = "select lid,content,`date`,fav,u.uid,u.username,u.face_url" +
		" from level l join user u on l.author=u.uid where l.thread=?"
	rows, err := da.Db.Query(sqlStr, tid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	for rows.Next() {
		l := model.Level{}
		err = rows.Scan(&l.Lid, &l.Content, &l.Date, &l.Fav, &l.Author.Uid, &l.Author.Username, &l.Author.FaceUrl)
		if err != nil {
			return nil, ec.MysqlErr
		}
		levels = append(levels, l)
	}
	t.Levels = levels
	return t, nil
}

func (ts *ThreadService) GetReply(_ *gin.Context, lid int) ([]*model.Reply, *ec.E) {
	replies := make([]*model.Reply, 0)
	sqlStr := "select rid,content,`date`,fav,author,`to`" +
		" from reply where level=?"
	rows, err := da.Db.Query(sqlStr, lid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	for rows.Next() {
		r := model.Reply{}
		var a, to int
		err = rows.Scan(&r.Rid, &r.Content, &r.Date, &r.Fav, &a, &to)
		if err != nil {
			return nil, ec.MysqlErr
		}
		author, err := S().Us.getUserByUid(a)
		if err != nil {
			return nil, ec.MysqlErr
		}
		toAuthor, err := S().Us.getUserByUid(to)
		if err != nil {
			return nil, ec.MysqlErr
		}
		r.Author = model.Author{
			Uid:      author.Uid,
			Username: author.Username,
			FaceUrl:  author.FaceUrl,
		}
		r.ToAuthor = model.Author{
			Uid:      toAuthor.Uid,
			Username: toAuthor.Username,
			FaceUrl:  toAuthor.FaceUrl,
		}
		replies = append(replies, &r)
	}
	return replies, nil
}

func (ts *ThreadService) CreateLevel(c *gin.Context, tx *sql.Tx, tid int, content string) (*model.Level, *ec.E) {
	g, _ := c.Get("uid")
	uid := g.(int)
	sqlStr := "insert into level(content,`date`,fav,thread,author,is_root) " +
		"values(?,?,?,?,?,?)"
	now := time.Now().Format("2006-01-02 15:04:05")
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.Exec(sqlStr, content, now, 0, tid, uid, true)
	} else {
		res, err = da.Db.Exec(sqlStr, content, now, 0, tid, uid, false)
	}
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	lid, _ := res.LastInsertId()
	author, err := s.Us.getUserByUid(uid)
	if err != nil {
		return nil, ec.MysqlErr
	}
	return &model.Level{
		Lid:     int(lid),
		Content: content,
		Date:    now,
		Fav:     0,
		Author: model.Author{
			Uid:      author.Uid,
			Username: author.Username,
			FaceUrl:  author.FaceUrl,
		},
		Replies: nil,
	}, nil
}

func (ts *ThreadService) CreateThread(c *gin.Context, title, content string) (*model.Thread, *ec.E) {
	g, _ := c.Get("uid")
	uid := g.(int)
	sqlStr := "insert into thread(title,`date`,fav,reply_num,last_modify,author)" +
		" values(?,?,?,?,?,?)"
	now := time.Now().Format("2006-01-02 15:04:05")
	tx, err := da.Db.Begin()
	if err != nil {
		log.Printf("begin trx error:%v", err)
		return nil, ec.MysqlErr
	}
	res, err := tx.Exec(sqlStr, title, now, 0, 0, now, uid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		_ = tx.Rollback()
		return nil, ec.MysqlErr
	}
	tid, _ := res.LastInsertId()
	author, err := s.Us.getUserByUid(uid)
	if err != nil {
		_ = tx.Rollback()
		return nil, ec.MysqlErr
	}
	if level, e := ts.CreateLevel(c, tx, int(tid), content); e != nil {
		_ = tx.Rollback()
		return nil, ec.MysqlErr
	} else {
		if err = tx.Commit(); err != nil {
			log.Printf("commit trx error:%v", err)
			return nil, ec.MysqlErr
		}
		return &model.Thread{
			Tid:        int(tid),
			Title:      title,
			Date:       now,
			Fav:        0,
			ReplyNum:   1,
			LastModify: now,
			Author: model.Author{
				Uid:      author.Uid,
				Username: author.Username,
				FaceUrl:  author.FaceUrl,
			},
			Levels: []model.Level{*level},
		}, nil
	}
}

func (ts *ThreadService) CreateReply(c *gin.Context, lid int, content string, to int) (*model.Reply, *ec.E) {
	g, _ := c.Get("uid")
	uid := g.(int)
	sqlStr := "insert into reply(content,`date`,fav,level,author,to)" +
		" values(?,?,?,?,?,?)"
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := da.Db.Exec(sqlStr, content, now, 0, lid, uid, to)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	rid, _ := res.LastInsertId()
	author, err := S().Us.getUserByUid(uid)
	if err != nil {
		return nil, ec.MysqlErr
	}
	toAuthor, err := S().Us.getUserByUid(to)
	if err != nil {
		return nil, ec.MysqlErr
	}
	return &model.Reply{
		Rid:     int(rid),
		Content: content,
		Date:    now,
		Fav:     0,
		ToAuthor: model.Author{
			Uid:      toAuthor.Uid,
			Username: toAuthor.Username,
			FaceUrl:  toAuthor.FaceUrl,
		},
		Author: model.Author{
			Uid:      author.Uid,
			Username: author.Username,
			FaceUrl:  author.FaceUrl,
		},
	}, nil
}

func (ts *ThreadService) DelReply(c *gin.Context, rid int) *ec.E {
	g, _ := c.Get("uid")
	uid := g.(int)
	row := da.Db.QueryRow("select author from sp_forum.reply where rid=?", rid)
	var author int
	if err := row.Scan(&author); err != nil {
		return ec.MysqlErr
	}
	if author != uid {
		return ec.NoAuth
	}
	sqlStr := "delete from sp_forum.reply where rid=?"
	_, err := da.Db.Exec(sqlStr, rid)
	if err != nil {
		return ec.MysqlErr
	}
	return nil
}

func (ts *ThreadService) DelLevel(c *gin.Context, lid int) *ec.E {
	g, _ := c.Get("uid")
	uid := g.(int)
	row := da.Db.QueryRow("select author from sp_forum.level where lid=?", lid)
	var author int
	if err := row.Scan(&author); err != nil {
		return ec.MysqlErr
	}
	if author != uid {
		return ec.NoAuth
	}

	tx, err := da.Db.Begin()
	if err != nil {
		log.Printf("begin trx error:%v", err)
		return ec.MysqlErr
	}
	sqlStr := "delete from sp_forum.level where lid=?"
	_, err = tx.Exec(sqlStr, lid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		_ = tx.Rollback()
		return ec.MysqlErr
	}
	sqlStr = "delete from sp_forum.reply where level=?"
	_, err = tx.Exec(sqlStr, lid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		_ = tx.Rollback()
		return ec.MysqlErr
	}
	if err = tx.Commit(); err != nil {
		log.Printf("commit trx error:%v", err)
		return ec.MysqlErr
	}
	return nil
}

func (ts *ThreadService) DelThread(c *gin.Context, tid int) *ec.E {
	g, _ := c.Get("uid")
	uid := g.(int)
	row := da.Db.QueryRow("select author from sp_forum.thread where tid=?", tid)
	var author int
	if err := row.Scan(&author); err != nil {
		return ec.MysqlErr
	}
	if author != uid {
		return ec.NoAuth
	}
	conn := da.OpenRedis()
	defer conn.Close()
	conn.Do("zrem", "thread:rank", tid)
	_, _ = conn.Do("zrem", "thread:liked", tid)
	thread, e := ts.GetThread(c, tid)
	if e != nil {
		return e
	}
	tx, err := da.Db.Begin()
	if err != nil {
		log.Printf("begin trx error:%v", err)
		return ec.MysqlErr
	}
	sqlStr := "delete from sp_forum.thread where tid=?"
	_, err = tx.Exec(sqlStr, tid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		_ = tx.Rollback()
		return ec.MysqlErr
	}
	sqlStr = "delete from sp_forum.level where thread=?"
	_, err = tx.Exec(sqlStr, tid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		_ = tx.Rollback()
		return ec.MysqlErr
	}
	in := ""
	for _, l := range thread.Levels {
		in += strconv.Itoa(l.Lid) + ","
	}
	in = in[:len(in)-1]
	sqlStr = "delete from sp_forum.reply where level in " +
		"(%s)"
	_, err = tx.Exec(fmt.Sprintf(sqlStr, in), tid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		_ = tx.Rollback()
		return ec.MysqlErr
	}
	if err = tx.Commit(); err != nil {
		log.Printf("commit trx error:%v", err)
		return ec.MysqlErr
	}
	return nil
}

func (ts *ThreadService) IsFav(c *gin.Context, lid int) (bool, *ec.E) {
	g, _ := c.Get("uid")
	uid := g.(int)
	conn := da.OpenRedis()
	defer conn.Close()
	res, err := redis.Bool(conn.Do("exists", "user:like:"+strconv.Itoa(uid)+":"+strconv.Itoa(lid)))
	if err != nil {
		return false, ec.RedisErr
	}
	return res, nil
}

func (ts *ThreadService) Fav(c *gin.Context, tid, lid int, positive bool) *ec.E {
	g, _ := c.Get("uid")
	uid := g.(int)
	conn := da.OpenRedis()
	defer conn.Close()
	var err error
	now := time.Now().Format("2006-01-02")
	isFav, e := ts.IsFav(c, lid)
	if e != nil {
		return e
	}
	if positive && !isFav {
		_ = conn.Send("multi")
		_ = conn.Send("set", "user:like:"+strconv.Itoa(uid)+":"+strconv.Itoa(lid), now)
		_ = conn.Send("zincrby", "thread:liked", 1, tid)
		_ = conn.Send("zincrby", "thread:rank", 1, tid)
		if _, err = conn.Do("exec"); err != nil {
			log.Printf("redis trx error:%v", err)
			return ec.RedisErr
		}
	} else if !positive && isFav {
		favDate, _ := redis.String(conn.Do("get", "user:like:"+strconv.Itoa(uid)+":"+strconv.Itoa(lid)))
		_, _ = conn.Do("zincrby", "thread:liked", -1, tid)
		favTime, _ := time.Parse("2006-01-02", favDate)
		diff := (time.Now().Unix() - favTime.Unix()) / 86400
		score, _ := redis.Float64(conn.Do("zincrby", "thread:rank", -math.Pow(0.75, float64(diff)), tid))
		if score <= 0 {
			_, _ = conn.Do("zrem", "level:rank", lid)
		}
		_, _ = conn.Do("del", "user:like:"+strconv.Itoa(uid)+":"+strconv.Itoa(lid))
	} else {
		return ec.ParamsErr
	}
	return nil
}

func (ts *ThreadService) FavNum(_ *gin.Context, lid int) (int, *ec.E) {
	conn := da.OpenRedis()
	defer conn.Close()
	favNum, err := redis.Int(conn.Do("zscore", "level:liked", lid))
	if err != nil {
		return 0, ec.RedisErr
	}
	return favNum, nil
}

func (ts *ThreadService) IsCollect(c *gin.Context, tid int) (bool, *ec.E) {
	g, _ := c.Get("uid")
	uid := g.(int)
	row := da.Db.QueryRow("select uid,tid from sp_forum.user_collect where uid=? and tid=?",
		uid, tid)
	var u, t int
	err := row.Scan(&u, &t)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, ec.MysqlErr
	}
	return true, nil
}

func (ts *ThreadService) Collect(c *gin.Context, tid int, positive bool) *ec.E {
	g, _ := c.Get("uid")
	uid := g.(int)
	isCollect, e := ts.IsCollect(c, tid)
	if e != nil {
		return e
	}
	var err error
	if isCollect && !positive {
		_, err = da.Db.Exec("delete from sp_forum.user_collect where uid=? and tid=?",
			uid, tid)
	} else if !isCollect && positive {
		_, err = da.Db.Exec("insert into sp_forum.user_collect(uid,tid) value(?,?)",
			uid, tid)
	}
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return ec.MysqlErr
	}
	return nil
}
