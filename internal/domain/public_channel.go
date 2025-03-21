package domain

import "github.com/gotd/td/tg"

type PublicChannel struct {
	ID             int64
	AccessHash     int64
	Title          string
	DiscussionPeer tg.InputPeerClass
}

func NewPublicChannel(id, accessHash int64, title string) *PublicChannel {
	return &PublicChannel{
		ID:         id,
		AccessHash: accessHash,
		Title:      title,
	}
}
