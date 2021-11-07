package model

type E struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
type RSP struct {
	Payload interface{} `json:"payload"`
	E
}

var OK = &E{0000, "OK"}
var MysqlErr = &E{0001, "数据库错误"}
var ParamsErr = &E{0002, "参数错误"}
var RedisErr = &E{1003, "获取缓存错误"}

var UsernameExist = &E{1001, "用户名已被注册"}
var PwdInvalid = &E{1002, "密码必须是8~18位的字母与数字组合"}
var UnOrPwdWrong = &E{1003, "用户名或密码错误"}
var GenTokenErr = &E{1004, "生成令牌错误"}
var NotLogin = &E{1005, "未登录"}
var UserNotExist = &E{1006, "用户不存在"}
var PwdNotMatch = &E{1007, "两次输入的密码不匹配"}
var DescTooLong = &E{1008, "简介不能超过110字"}
var OldPwdWrong = &E{1009, "旧密码输入错误"}
var NoAuth = &E{1010, "没有权限"}
var FollowSelf = &E{1011, "不能关注自己"}

var ThreadNotExist = &E{2001, "帖子不存在"}
