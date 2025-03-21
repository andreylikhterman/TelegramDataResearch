package domain

type Message struct {
	Text         string
	Comment_id   int
	Post_id      int
	User_id      int
	User_name    string
	Channel_name string
	Channel_id   int64
}
