package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func (d *MyDB) InsertChannel(id int64, title string, subscribers int, channelType string) {
	const query = `
    INSERT INTO channels (id, title, type, subscribers_counter)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (id) DO NOTHING;
    `
	if _, err := d.DB.Exec(query, id, title, channelType, subscribers); err != nil {
		log.Fatalf("Failed to insert channel: %v", err)
	}
}

func (d *MyDB) InsertMessage(args []any) {
	const batchSize = 10
	if len(args) != batchSize*9 {
		log.Fatalf("Expected %d arguments for InsertMessage, got %d", batchSize*9, len(args))
	}

	placeholders := make([]string, batchSize)
	for i := range batchSize {
		start := i*9 + 1
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			start, start+1, start+2, start+3, start+4, start+5, start+6, start+7, start+8)
	}

	query := fmt.Sprintf(`
INSERT INTO comments (tg_id, post_id, replied_to, user_id, timestamp, value)
SELECT tmp.id::bigint, p.id, tmp.replied_to::bigint, tmp.user_id::bigint, tmp.data::time, tmp.value
FROM (VALUES %s) AS tmp(value, id, post_id, user_id, user_name, channel_name, channel_id, data, replied_to)
LEFT JOIN posts p ON p.post_id_into_channel = tmp.post_id::bigint
WHERE p.id IS NOT NULL;
`, strings.Join(placeholders, ", "))

	if _, err := d.DB.Exec(query, args...); err != nil {
		log.Fatalf("Failed to insert messages: %v", err)
	}
}

func (d *MyDB) GetMaxCommentIDByChannel(channelID int64) (int64, error) {
	const query = `
        SELECT COALESCE(MAX(c.tg_id), 0)
        FROM comments c
        JOIN posts p ON c.post_id = p.id
        WHERE p.channel_id = $1;
    `
	var maxID sql.NullInt64
	err := d.DB.QueryRow(query, channelID).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	if maxID.Valid {
		return maxID.Int64, nil
	}
	return 0, nil
}

func (d *MyDB) InsertUser(id int64, name string) {
	const query = `
    INSERT INTO users (id, name)
    VALUES ($1, $2)
    ON CONFLICT (id) DO NOTHING;
    `
	if _, err := d.DB.Exec(query, id, name); err != nil {
		log.Fatalf("Failed to insert user: %v", err)
	}
}

func (d *MyDB) InsertPost(placeholders []string, args []any) {
	query := fmt.Sprintf(`
INSERT INTO posts (channel_id, post_id_into_channel, timestamp, value)
SELECT tmp.channel_id::bigint, tmp.post_id_into_channel::bigint, tmp.data::time, tmp.name
FROM (VALUES %s) AS tmp(name, post_id_into_channel, channel_id, channel, data);
`, strings.Join(placeholders, ", "))

	if _, err := d.DB.Exec(query, args...); err != nil {
		log.Fatalf("Failed to insert posts: %v", err)
	}
}
