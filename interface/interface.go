package main

import (
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jmoiron/sqlx"
	"log"
	"sp-forum-back/model"
	ec "sp-forum-back/model"
	"sp-forum-back/service"
	"sp-forum-back/utils"
	"strconv"
	"strings"
	"time"
)

var (
	s = service.S()
)

func main() {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:7999")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Expose-Headers", "Authorization")
	})
	r.Use(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.String(200, "")
		}
	})
	auth := func(c *gin.Context) {
		au := c.GetHeader("Authorization")
		if au != "" {
			claims, err := utils.ParseToken(au)
			if err != nil {
				log.Printf("parse token error:%v", err)
				goto abort
			}
			exp := claims.ExpiresAt
			now := time.Now().Unix()
			if now > exp {
				log.Printf("token expired")
				goto abort
			}
			c.Set("uid", claims.Uid)
			c.Next()
			token, err := utils.GenerateToken(claims.Uid)
			if err == nil {
				c.Header("Authorization", token)
				return
			}
		}
		goto abort
	abort:
		c.JSON(200, ec.NotLogin)
		c.Abort()
	}

	r.GET("/auth", auth)
	r.POST("/user/register", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		password2 := c.PostForm("password2")
		ud, err := s.Us.Register(c, username, password, password2)
		if err != nil {
			c.JSON(200, &err)
		} else {
			c.JSON(200, &ec.RSP{Payload: ud})
		}
	})
	r.POST("/user/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")
		ud, err := s.Us.Login(c, username, password)
		if err != nil {
			c.JSON(200, &err)
		} else {
			c.JSON(200, &ec.RSP{Payload: ud})
		}
	})
	r.POST("/user/changeProfile", auth, func(c *gin.Context) {
		req := model.ChangeUserProfileReq{}
		if err := c.Bind(&req); err != nil {
			log.Printf("bind json error:%v", err)
			c.JSON(200, ec.ParamsErr)
			return
		}
		err := s.Us.ChangeProfile(c, req)
		if err != nil {
			c.JSON(200, &err)
		} else {
			c.JSON(200, &ec.OK)
		}
	})
	r.POST("/user/follow", auth, func(c *gin.Context) {
		t := c.PostForm("target")
		target, _ := strconv.Atoi(t)
		p := c.PostForm("positive")
		e := s.Us.Follow(c, target, p == "true")
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.OK)
		}
	})
	r.GET("/user/isFollow", auth, func(c *gin.Context) {
		t := c.Query("target")
		target, _ := strconv.Atoi(t)
		isFollow, e := s.Us.IsFollow(c, target)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: isFollow})
		}
	})
	r.GET("/user/stats", auth, func(c *gin.Context) {
		stats, e := s.Us.Stats(c)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: stats})
		}
	})
	r.GET("/thread/getThread", func(c *gin.Context) {
		t := c.Query("tid")
		p := c.Query("page")
		tid, _ := strconv.Atoi(t)
		page, _ := strconv.Atoi(p)
		thread, e := s.Ts.GetThread(c, tid, page)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: thread})
		}
	})
	r.GET("/thread/levelNum", func(c *gin.Context) {
		t := c.Query("tid")
		tid, _ := strconv.Atoi(t)
		cnt, e := s.Ts.LevelNum(c, tid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: cnt})
		}
	})
	r.GET("/thread/getReply", func(c *gin.Context) {
		l := c.Query("lid")
		lid, _ := strconv.Atoi(l)
		replies, e := s.Ts.GetReply(c, lid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: replies})
		}
	})
	r.POST("/thread/createThread", func(c *gin.Context) {
		title := c.PostForm("title")
		content := c.PostForm("content")
		thread, e := s.Ts.CreateThread(c, title, content)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: thread})
		}
	})
	r.POST("/thread/createLevel", func(c *gin.Context) {
		t := c.PostForm("tid")
		tid, _ := strconv.Atoi(t)
		content := c.PostForm("content")
		level, e := s.Ts.CreateLevel(c, nil, tid, content)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: level})
		}
	})
	r.POST("/thread/createReply", func(c *gin.Context) {
		l := c.PostForm("lid")
		lid, _ := strconv.Atoi(l)
		content := c.PostForm("content")
		t := c.PostForm("to")
		to, _ := strconv.Atoi(t)
		reply, e := s.Ts.CreateReply(c, lid, content, to)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: reply})
		}
	})
	r.POST("/thread/delReply", auth, func(c *gin.Context) {
		r := c.PostForm("rid")
		rid, _ := strconv.Atoi(r)
		e := s.Ts.DelReply(c, rid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.OK)
		}
	})
	r.POST("/thread/delLevel", auth, func(c *gin.Context) {
		l := c.PostForm("lid")
		lid, _ := strconv.Atoi(l)
		e := s.Ts.DelReply(c, lid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.OK)
		}
	})
	r.POST("/thread/delThread", auth, func(c *gin.Context) {
		t := c.PostForm("tid")
		tid, _ := strconv.Atoi(t)
		e := s.Ts.DelReply(c, tid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.OK)
		}
	})
	r.GET("/thread/isFav", auth, func(c *gin.Context) {
		l := c.PostForm("lid")
		lid, _ := strconv.Atoi(l)
		isFav, e := s.Ts.IsFav(c, lid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: isFav})
		}
	})
	r.POST("/thread/fav", auth, func(c *gin.Context) {
		t := c.PostForm("tid")
		tid, _ := strconv.Atoi(t)
		l := c.PostForm("lid")
		lid, _ := strconv.Atoi(l)
		p := c.PostForm("positive")
		log.Printf("fav positive:%v", p)
		e := s.Ts.Fav(c, tid, lid, p == "true")
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.OK)
		}
	})
	r.GET("/thread/favNum", func(c *gin.Context) {
		l := c.PostForm("lid")
		lid, _ := strconv.Atoi(l)
		favNum, e := s.Ts.FavNum(c, lid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: favNum})
		}
	})
	r.GET("/thread/isCollect", auth, func(c *gin.Context) {
		t := c.PostForm("tid")
		tid, _ := strconv.Atoi(t)
		isCollect, e := s.Ts.IsCollect(c, tid)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: isCollect})
		}
	})
	r.POST("/thread/collect", auth, func(c *gin.Context) {
		t := c.PostForm("tid")
		tid, _ := strconv.Atoi(t)
		p := c.PostForm("positive")
		e := s.Ts.Collect(c, tid, p == "true")
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.OK)
		}
	})

	r.GET("/home/hots", func(c *gin.Context) {
		hots, e := s.Hs.Hots(c)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: hots})
		}
	})
	r.GET("/home/carousel", func(c *gin.Context) {
		carousel, e := s.Hs.Carousel(c)
		if e != nil {
			c.JSON(200, &e)
		} else {
			c.JSON(200, &ec.RSP{Payload: carousel})
		}
	})

	r.POST("/uploadImg", auth, func(c *gin.Context) {
		type Data struct {
			Url string `json:"url"`
		}
		type uploadImgRsp struct {
			Errno int    `json:"errno"`
			Data  []Data `json:"data"`
		}
		form, err := c.MultipartForm()
		if err != nil {
			log.Printf("recv file error:%v", err)
			c.JSON(200, uploadImgRsp{Errno: -1})
			return
		}
		retData := make([]Data, 0)
		for _, img := range form.File["myFile"] {
			filename := img.Filename
			ts := time.Now().Unix()
			g, _ := c.Get("uid")
			uid := g.(int)
			fullName := strings.Join([]string{strconv.Itoa(uid), strconv.Itoa(int(ts)), filename}, "_")
			log.Printf("fullName:%v", fullName)
			err = c.SaveUploadedFile(img, "D:\\nginx-1.21.4\\html\\dist\\img\\"+fullName)
			if err != nil {
				log.Printf("save img error:%v", err)
				c.JSON(200, uploadImgRsp{Errno: -1})
				return
			}
			retData = append(retData, Data{Url: "localhost:7999/img/" + fullName})
		}

		c.JSON(200, uploadImgRsp{Errno: 0, Data: retData})
	})

	_ = r.Run(":8999")
}
