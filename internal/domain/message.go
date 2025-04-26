package domain

type Message struct {
	Text        string
	CommentId   int
	PostId      int
	UserId      int
	UserName    string
	ChannelName string
	ChannelId   int64
}
