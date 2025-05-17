package domain

import "time"

type Message struct {
	Text        string
	CommentId   int
	PostId      int
	UserId      int
	UserName    string
	ChannelId   int64
	ChannelName string
	Data        time.Time
	RepliedTo   int
}
