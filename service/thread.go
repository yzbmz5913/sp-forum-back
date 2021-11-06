package service

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
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

func (ts *ThreadService) DelReply(_ *gin.Context, rid int) *ec.E {
	sqlStr := "delete from sp_forum.reply where rid=?"
	_, err := da.Db.Exec(sqlStr, rid)
	if err != nil {
		return ec.MysqlErr
	}
	return nil
}

func (ts *ThreadService) DelLevel(_ *gin.Context, lid int) *ec.E {
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

func (ts *ThreadService) isFav(c *gin.Context, lid, rid int) *ec.E {
	panic("implement me")
}

func (ts *ThreadService) Fav(c *gin.Context, lid, rid int) *ec.E {
	panic("implement me")
}

func (ts *ThreadService) Collect(c *gin.Context, tid int) *ec.E {
	panic("implement me")
}