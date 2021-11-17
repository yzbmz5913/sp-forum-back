package service

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"log"
	"math/rand"
	"reflect"
	"sp-forum-back/da"
	"sp-forum-back/model"
	"sp-forum-back/utils"
	"strconv"
	"time"
)
import ec "sp-forum-back/model"

type UserService struct{}

func (us *UserService) Register(c *gin.Context, username, password, password2 string) (*model.UserDetail, *ec.E) {
	row := da.Db.QueryRow("select `username` from sp_forum.user where `username`=?", username)
	var un string
	err := row.Scan(&un)
	if err != nil {
		if err == sql.ErrNoRows {
			if !utils.CheckPassword(password) {
				return nil, ec.PwdInvalid
			}
			if password2 != password {
				return nil, ec.PwdNotMatch
			}
			desc := "小镇普通居民一枚"
			var faces = []string{
				"https://img0.baidu.com/it/u=668882205,3932911443&fm=26&fmt=auto",
				"https://img0.baidu.com/it/u=216031677,97581716&fm=15&fmt=auto",
				"https://img0.baidu.com/it/u=3295979978,4165272095&fm=26&fmt=auto",
			}
			idx := rand.Int31n(int32(len(faces)))
			faceUrl := faces[idx]
			regDate := time.Now().Format("2006-01-02 15:04")
			var m = md5.New()
			res, err := da.Db.Exec("insert into `user`(`username`, `password`, `desc`, `face_url`,`reg_date`) value (?,?,?,?,?)",
				username, m.Sum([]byte(password)), desc, faceUrl, regDate)
			if err != nil {
				log.Printf("exec mysql error:%v", err)
				return nil, ec.MysqlErr
			}
			uid, _ := res.LastInsertId()
			token, err := utils.GenerateToken(int(uid))
			if err == nil {
				c.Header("Authorization", token)
			}
			return &model.UserDetail{
				Uid:      int(uid),
				Username: username,
				Desc:     desc,
				FaceUrl:  faceUrl,
				RegDate:  regDate,
			}, nil
		} else {
			log.Fatalf("exec mysql error:%v", err)
			return nil, ec.MysqlErr
		}
	} else {
		return nil, ec.UsernameExist
	}
}

func (us *UserService) Login(c *gin.Context, username, password string) (*model.UserDetail, *ec.E) {
	au := c.GetHeader("Authorization")
	if au != "" {
		claims, err := utils.ParseToken(au)
		if err != nil {
			log.Printf("parse token error:%v", err)
			goto do
		}
		exp := claims.ExpiresAt
		now := time.Now().Unix()
		if now > exp {
			log.Printf("token expired")
			goto do
		}
		ud, err := us.getUserByUid(claims.Uid)
		token, err := utils.GenerateToken(claims.Uid)
		if err == nil {
			c.Header("Authorization", token)
		}
		return ud, nil
	}
do:
	var m = md5.New()
	row := da.Db.QueryRow("select `uid`,`username`,`desc`,`face_url` from `user` where `username`=? and `password`=?", username, m.Sum([]byte(password)))
	var ud = &model.UserDetail{}
	err := row.Scan(&ud.Uid, &ud.Username, &ud.Desc, &ud.FaceUrl)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ec.UnOrPwdWrong
		}
		log.Printf("login err:%v", err)
		return nil, ec.MysqlErr
	}
	token, err := utils.GenerateToken(ud.Uid)
	if err != nil {
		log.Printf("gen token error:%v", err)
		return nil, ec.GenTokenErr
	}
	c.Header("Authorization", token)
	return ud, nil
}

func (us *UserService) ChangeProfile(c *gin.Context, req model.ChangeUserProfileReq) *ec.E {
	get, _ := c.Get("uid")
	uid := get.(int)
	_, err := us.getUserByUid(uid)
	if err != nil {
		return ec.UserNotExist
	}
	if len([]rune(req.Desc)) > 110 {
		return ec.DescTooLong
	}

	sqlStr := "update `user` set `username`=?, `desc`=?, `face_url`=? %s " +
		"where `uid`=?"
	args := make([]interface{}, 0, 4)
	args = append(args, req.Username)
	args = append(args, req.Desc)
	args = append(args, req.FaceUrl)
	if req.NewPwd != "" {
		row := da.Db.QueryRow("select `password` from user where `uid`=?", uid)
		var oldPwdMd5 []byte
		_ = row.Scan(&oldPwdMd5)
		m := md5.New()
		m.Sum([]byte(req.OldPwd))
		if !reflect.DeepEqual(oldPwdMd5, m.Sum([]byte(req.OldPwd))) {
			return ec.OldPwdWrong
		}
		if !utils.CheckPassword(req.NewPwd) {
			return ec.PwdInvalid
		}
		m.Reset()
		args = append(args, m.Sum([]byte(req.NewPwd)))
		sqlStr = fmt.Sprintf(sqlStr, ", `password`=?")
	} else {
		sqlStr = fmt.Sprintf(sqlStr, "")
	}
	args = append(args, uid)

	if _, err = da.Db.Exec(sqlStr, args...); err != nil {
		return ec.MysqlErr
	}
	if err = utils.Retry(3, time.Second, func() error {
		conn := da.OpenRedis()
		defer conn.Close()
		_, err := conn.Do("del", "user_"+strconv.Itoa(uid))
		return err
	}); err != nil {
		log.Printf("retry error:%v", err)
		return ec.RedisErr
	}
	return nil
}

func (us *UserService) getUserByUid(uid int) (*model.UserDetail, error) {
	conn := da.OpenRedis()

	defer func(conn redis.Conn) {
		_ = conn.Close()
	}(conn)
	userJson, err := redis.Bytes(conn.Do("get", "user_"+strconv.Itoa(uid)))
	if err == nil {
		ud := model.UserDetail{}
		if err := json.Unmarshal(userJson, &ud); err != nil {
			log.Printf("unmarshal json error:%v", err)
		} else {
			return &ud, nil
		}
	}

	row := da.Db.QueryRow("select `uid`,`username`,`desc`,`face_url`,`reg_date` from `user` where `uid`=?", uid)
	ud := model.UserDetail{}
	err = row.Scan(&ud.Uid, &ud.Username, &ud.Desc, &ud.FaceUrl, &ud.RegDate)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("user not exists")
			return nil, err
		}
		return nil, err
	}
	bytes, err := json.Marshal(ud)
	if err == nil {
		_ = conn.Send("setex", "user_"+strconv.Itoa(uid), 86400, bytes)
	}
	return &ud, nil
}

//Follow positive=true关注 positive=false取关
func (us *UserService) Follow(c *gin.Context, targetUid int, positive bool) *ec.E {
	get, _ := c.Get("uid")
	uid := get.(int)
	if uid == targetUid {
		return ec.FollowSelf
	}
	var err error
	if positive {
		sqlStr := "insert into sp_forum.following(uid1,uid2) value(?,?)"
		_, err = da.Db.Exec(sqlStr, uid, targetUid)
		_, _ = da.Db.Exec("insert into notification(uid,`from`,type,date) value(?,?,?,?)",
			targetUid, uid, 1, time.Now().Format("2006-01-02 15:04:05"))
	} else {
		sqlStr := "delete from sp_forum.following where uid1=? and uid2=?"
		_, err = da.Db.Exec(sqlStr, uid, targetUid)
	}
	if err != nil {
		return ec.MysqlErr
	}
	return nil
}

func (us *UserService) IsFollow(c *gin.Context, targetUid int) (bool, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	sqlStr := "select uid1,uid2 from sp_forum.following where uid1=? and uid2=?"
	row := da.Db.QueryRow(sqlStr, uid, targetUid)
	var uid1, uid2 int
	err := row.Scan(&uid1, &uid2)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, ec.MysqlErr
	}
	return true, nil
}

func (us *UserService) Stats(c *gin.Context) (*model.UserStats, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	row := da.Db.QueryRow("select reg_date from user where uid=?", uid)
	var regDate string
	if err := row.Scan(&regDate); err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	row = da.Db.QueryRow("select count(1) from thread where author=?", uid)
	var threadNum int
	if err := row.Scan(&threadNum); err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	row = da.Db.QueryRow("select count(1) from following where uid1=?", uid)
	var followingNum int
	if err := row.Scan(&followingNum); err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	row = da.Db.QueryRow("select count(1) from following where uid2=?", uid)
	var followerNum int
	if err := row.Scan(&followerNum); err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	return &model.UserStats{
		RegDate:      regDate,
		ThreadNum:    threadNum,
		FollowingNum: followingNum,
		FollowerNum:  followerNum,
	}, nil
}

func (us *UserService) History(c *gin.Context) ([]*model.Post, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	rows, err := da.Db.Query("select tid,date from visit where uid=? order by date desc", uid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	posts := make([]*model.Post, 0)
	for rows.Next() {
		var tid int
		var date string
		if err := rows.Scan(&tid, &date); err != nil {
			continue
		}
		p := &model.Post{
			Author: model.Author{},
		}
		row := da.Db.QueryRow("select tid,title,u.uid,u.username,u.face_url from thread t join user u on t.author=u.uid where tid=?", tid)
		if err = row.Scan(&p.Tid, &p.Title, &p.Author.Uid, &p.Author.Username, &p.Author.FaceUrl); err == nil {
			if e := PostDetail(p); e == nil {
				p.LastModify = date
				posts = append(posts, p)
			}
		}
	}
	return posts, nil
}

func (us *UserService) Posts(c *gin.Context) ([]*model.Post, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	rows, err := da.Db.Query("select tid,title,last_modify,u.uid,u.username,u.face_url from thread t join user u on t.author=u.uid where u.uid=?", uid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	posts := make([]*model.Post, 0)
	for rows.Next() {
		p := &model.Post{
			Author: model.Author{},
		}
		if err := rows.Scan(&p.Tid, &p.Title, &p.LastModify, &p.Author.Uid, &p.Author.Username, &p.Author.FaceUrl); err == nil {
			if e := PostDetail(p); e == nil {
				posts = append(posts, p)
			}
		}
	}
	return posts, nil
}

func (us *UserService) Collection(c *gin.Context) ([]*model.Post, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	rows, err := da.Db.Query("select tid from user_collect where uid=?", uid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	posts := make([]*model.Post, 0)
	for rows.Next() {
		var tid int
		if err := rows.Scan(&tid); err != nil {
			continue
		}
		p := &model.Post{
			Author: model.Author{},
		}
		row := da.Db.QueryRow("select tid,title,last_modify,u.uid,u.username,u.face_url from thread t join user u on t.author=u.uid where tid=?", tid)
		if err = row.Scan(&p.Tid, &p.Title, &p.LastModify, &p.Author.Uid, &p.Author.Username, &p.Author.FaceUrl); err == nil {
			if e := PostDetail(p); e == nil {
				posts = append(posts, p)
			}
		}
	}
	return posts, nil
}

func (us *UserService) Following(c *gin.Context) ([]*model.User, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	rows, err := da.Db.Query("select u.uid,u.username,u.`desc`,u.face_url from following join user u on uid2=u.uid where uid1=?", uid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	users := make([]*model.User, 0)
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.Uid, &u.Username, &u.Desc, &u.FaceUrl); err == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (us *UserService) Follower(c *gin.Context) ([]*model.User, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	rows, err := da.Db.Query("select u.uid,u.username,u.`desc`,u.face_url from following join user u on uid1=u.uid where uid2=?", uid)
	if err != nil {
		log.Printf("exec mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	users := make([]*model.User, 0)
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.Uid, &u.Username, &u.Desc, &u.FaceUrl); err == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (us *UserService) Notifications(c *gin.Context) ([]*model.Notification, *ec.E) {
	get, _ := c.Get("uid")
	uid := get.(int)
	nts := make([]*model.Notification, 0)
	rows, err := da.Db.Query("select u.uid,u.username,u.face_url,type,`date`,thread,content from notification join user u using(uid) where uid=?", uid)
	if err != nil {
		log.Printf("query mysql error:%v", err)
		return nil, ec.MysqlErr
	}
	for rows.Next() {
		nt := &model.Notification{From: model.Author{}}
		if err := rows.Scan(&nt.From.Uid, &nt.From.Username, &nt.From.FaceUrl, &nt.Type, &nt.Date, &nt.Tid, &nt.Content); err != nil {
			log.Printf("scan mysql error:%v", err)
			continue
		}
		if nt.Type == 2 || nt.Type == 3 {
			row := da.Db.QueryRow("select title from thread where tid=?", nt.Tid)
			var title string
			if err := row.Scan(&title); err != nil {
				log.Printf("scan mysql error:%v", err)
				continue
			}
			nt.Title = title
		}
		nts = append(nts, nt)
	}
	return nts, nil
}
