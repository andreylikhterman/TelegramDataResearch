package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

func (d *MyDB) InsertZeroPostIfNotExists() (int64, error) {
	var existingID int64

	// Сначала ищем, есть ли уже такая запись
	querySelect := `
        SELECT id
        FROM posts
        WHERE channel_id = 0
          AND post_id_into_channel = 0
          AND timestamp = '00:00:00'
          AND value = ''
        LIMIT 1
    `
	err := d.DB.QueryRow(querySelect).Scan(&existingID)
	if err == nil {
		return existingID, nil
	}
	if err != sql.ErrNoRows {

		return 0, fmt.Errorf("failed to check existing post: %w", err)
	}

	queryInsert := `
        INSERT INTO posts (channel_id, post_id_into_channel, timestamp, value)
        VALUES (0, 0, '00:00:00', '')
        RETURNING id
    `
	var insertedID int64
	err = d.DB.QueryRow(queryInsert).Scan(&insertedID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert zero post: %w", err)
	}

	return insertedID, nil
}

func (d *MyDB) InsertChannel(id int64, title string, count int, channelType string) {
	_, err := d.DB.Exec(`
    INSERT INTO channels (id, title, type, subscribers_counter)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (id) DO NOTHING;`,
		id, title, channelType, count)
	if err != nil {
		log.Fatalf("Failed to insert channel: %v", err)
	}
}

func (d *MyDB) InsertMessage(args []any) {
	placeholders := make([]string, 10)
	for i := 0; i < 10; i++ {
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*9+1, i*9+2, i*9+3, i*9+4, i*9+5, i*9+6, i*9+7, i*9+8, i*9+9)
	}
	query := fmt.Sprintf(`
INSERT INTO comments (tg_id, post_id, replied_to, user_id, timestamp, value)
SELECT tmp.id::bigint, p.id, tmp.replied_to::bigint, tmp.user_id::bigint, tmp.data::time, tmp.value
FROM (VALUES %s) AS tmp(value, id, post_id, user_id, user_name, channel_name, channel_id, data, replied_to)
LEFT JOIN posts p ON p.post_id_into_channel = tmp.post_id::bigint
WHERE p.id IS NOT NULL;
`, strings.Join(placeholders, ", "))
	result, err := d.DB.Exec(query, args...)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	_, _ = result.RowsAffected()
}

func (d *MyDB) GetMaxCommentIDByChannel(channelID int64) (int64, error) {
	var maxCommentID sql.NullInt64
	query := `
        SELECT COALESCE(MAX(c.tg_id), 0)
        FROM comments c
        JOIN posts p ON c.post_id = p.id
        WHERE p.channel_id = $1
    `
	err := d.DB.QueryRow(query, channelID).Scan(&maxCommentID)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}
	if maxCommentID.Valid {
		return maxCommentID.Int64, nil
	}
	return 0, nil
}

func (d *MyDB) InsertUser(id int64, name string) {
	_, err := d.DB.Exec(`
    INSERT INTO users (id, name)
    VALUES ($1, $2)
    ON CONFLICT (id) DO NOTHING;`,
		id, name)
	if err != nil {
		log.Fatalf("Failed to insert channel: %v", err)
	}
}

func (d *MyDB) InsertPost(placeholders []string, args []any) {
	query := fmt.Sprintf(`
INSERT INTO posts (channel_id, post_id_into_channel, timestamp, value)
SELECT tmp.channel_id::bigint, tmp.post_id_into_channel::bigint, tmp.data::time, tmp.name
FROM (VALUES %s) AS tmp(name, post_id_into_channel, channel_id, channel, data);
`, strings.Join(placeholders, ", "))
	result, err := d.DB.Exec(query, args...)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса с постами: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Добавлено постов: %d\n", rowsAffected)
}
