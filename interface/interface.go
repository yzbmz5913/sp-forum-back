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
	"time"
)

var (
	s = service.S()
)

func main() {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:8081")
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

	_ = r.Run()
}
