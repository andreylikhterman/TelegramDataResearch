package db

import (
	"fmt"
	"log"
	"strings"
)

func (d *MyDB) InsertChannel(id int64, title string) {
	_, err := d.DB.Exec(`
    INSERT INTO channels (id, title, type, subscribers_counter)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (id) DO NOTHING;`,
		id, title, "", 0)
	if err != nil {
		log.Fatalf("Failed to insert channel: %v", err)
	}
}

func (d *MyDB) InsertMessage(placeholders []string, args []any) {
	query := fmt.Sprintf(`
INSERT INTO comments (id, post_id, replied_to, user_id, timestamp, value)
SELECT tmp.id::bigint, p.id, 0, tmp.user_id::bigint, '0001-01-01 00:00:00', tmp.value
FROM (VALUES %s) AS tmp(value, id, post_id, user_id, user_name, channel_name, channel_id)
LEFT JOIN posts p ON p.post_id_into_channel = tmp.post_id::bigint
WHERE p.id IS NOT NULL;
`, strings.Join(placeholders, ", "))
	result, err := d.DB.Exec(query, args...)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Добавлено сообщений: %d\n", rowsAffected)
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
SELECT tmp.channel_id::bigint, tmp.post_id_into_channel::bigint, '0001-01-01 00:00:00', tmp.name
FROM (VALUES %s) AS tmp(name, post_id_into_channel, channel_id, channel);
`, strings.Join(placeholders, ", "))
	result, err := d.DB.Exec(query, args...)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса с постами: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Добавлено постов: %d\n", rowsAffected)
}
