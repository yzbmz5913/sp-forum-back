package model

type ChangeUserProfileReq struct {
	Username string `form:"username"`
	Desc     string `form:"desc"`
	FaceUrl  string `form:"face_url"`
	OldPwd   string `form:"old_pwd"`
	NewPwd   string `form:"new_pwd"`
}

type UserDetail struct {
	Uid      int    `json:"uid"`
	Username string `json:"username"`
	Password string `json:"password"`
	Desc     string `json:"desc"`
	FaceUrl  string `json:"face_url"`
	RegDate  string `json:"reg_date"`
}

type Author struct {
	Uid      int    `json:"uid"`
	Username string `json:"username"`
	FaceUrl  string `json:"face_url"`
}

type Thread struct {
	Tid        int     `json:"tid,omitempty"`
	Title      string  `json:"title,omitempty"`
	Date       string  `json:"date,omitempty"`
	Fav        int     `json:"fav,omitempty"`
	ReplyNum   int     `json:"reply_num,omitempty"`
	LastModify string  `json:"last_modify,omitempty"`
	Author     Author  `json:"author"`
	Levels     []Level `json:"levels"`
}

type Level struct {
	Lid     int     `json:"lid,omitempty"`
	Content string  `json:"content,omitempty"`
	Date    string  `json:"date,omitempty"`
	Fav     int     `json:"fav,omitempty"`
	Author  Author  `json:"author"`
	Replies []Reply `json:"replies"`
}

type Reply struct {
	Rid      int    `json:"rid,omitempty"`
	Content  string `json:"content,omitempty"`
	Date     string `json:"date,omitempty"`
	Fav      int    `json:"fav,omitempty"`
	ToAuthor Author `json:"to_author"`
	Author   Author `json:"author"`
}
