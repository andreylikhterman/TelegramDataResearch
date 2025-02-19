package domain

type PublicChannel struct {
	ID         int64
	AccessHash int64
	Title      string
}

func NewPublicChannel(id, accessHash int64, title string) *PublicChannel {
	return &PublicChannel{
		ID:         id,
		AccessHash: accessHash,
		Title:      title,
	}
}
