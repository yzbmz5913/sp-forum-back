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

type UserStats struct {
	RegDate      string `json:"reg_date"`
	ThreadNum    int    `json:"thread_num"`
	FollowingNum int    `json:"following_num"`
	FollowerNum  int    `json:"follower_num"`
}

type Author struct {
	Uid      int    `json:"uid"`
	Username string `json:"username"`
	FaceUrl  string `json:"face_url"`
}

type User struct {
	Author
	Desc string `json:"desc"`
}

type Thread struct {
	Tid        int     `json:"tid,omitempty"`
	Title      string  `json:"title,omitempty"`
	Date       string  `json:"date,omitempty"`
	LastModify string  `json:"last_modify,omitempty"`
	Author     Author  `json:"author"`
	Levels     []Level `json:"levels"`
}

type Level struct {
	Lid      int    `json:"lid,omitempty"`
	Content  string `json:"content,omitempty"`
	Date     string `json:"date,omitempty"`
	Fav      int    `json:"fav,omitempty"`
	IsRoot   bool   `json:"is_root,omitempty"`
	ReplyNum int    `json:"reply_num,omitempty"`
	Author   Author `json:"author"`
}

type Reply struct {
	Rid      int    `json:"rid,omitempty"`
	Content  string `json:"content,omitempty"`
	Date     string `json:"date,omitempty"`
	ToAuthor Author `json:"to_author"`
	Author   Author `json:"author"`
}

type Hot struct {
	Tid   int    `json:"tid,omitempty"`
	Title string `json:"title,omitempty"`
	Date  string `json:"date,omitempty"`
}

type Carousel struct {
	Tid        int    `json:"tid,omitempty"`
	Title      string `json:"title,omitempty"`
	PictureUrl string `json:"picture_url,omitempty"`
}

type Post struct {
	Tid        int    `json:"tid,omitempty"`
	Title      string `json:"title,omitempty"`
	Abstract   string `json:"abstract,omitempty"`
	Author     Author `json:"author"`
	ReplyNum   int    `json:"reply_num,omitempty"`
	VisitNum   int    `json:"visit_num,omitempty"`
	LastModify string `json:"last_modify,omitempty"`
}

type Notification struct {
	Type    int    `json:"type,omitempty"`
	From    Author `json:"from"`
	Date    string `json:"date,omitempty"`
	Content string `json:"content,omitempty"`
	Title   string `json:"title,omitempty"`
	Tid     int    `json:"tid,omitempty"`
}

const EsIndexName = "sp_forum_posts"

type EsPost struct {
	Title   string `json:"title,omitempty"`
	Author  string `json:"author,omitempty"`
	Content string `json:"content,omitempty"`
}
