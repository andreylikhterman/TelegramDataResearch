package domain

import "time"

type Post struct {
	Text        string
	PostId      int
	ChannelId   int64
	ChannelName string
	Data        time.Time
}
