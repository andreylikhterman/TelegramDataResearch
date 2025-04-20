package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goose "github.com/pressly/goose/v3"
)

//Что нужно скачать для локального запуска БД:
//sudo apt update
//sudo apt install postgresql postgresql-contrib
//sudo systemctl start postgresql
//
//Создадим пользователя для подключения к БД:
//sudo -u postgres psql -c "CREATE USER youruser WITH PASSWORD 'yourpass';"
//sudo -u postgres psql -c "ALTER USER youruser WITH PASSWORD 'yourpass';"
//sudo -u postgres psql -c "CREATE DATABASE yourdb OWNER youruser;"
//(Сервер постгреса поднимаетя там, где написано в файле конфигурации)
//
//Для подключения к БД и просмотру содержимого нужно обратиться командой:
//PGPASSWORD=yourpass psql "host=localhost port=5432 user=youruser password=yourpass dbname=yourdb sslmode=disable"

//Пример .env файла:
//DB_HOST=localhost
//DB_PORT=5432
//DB_USER=youruser
//DB_PASSWORD=yourpass
//DB_NAME=yourdb
//SSL_MODE=disableл

func Connect() *sql.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("SSL_MODE"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	_, b, _, _ := runtime.Caller(0)
	dir := filepath.Dir(b)
	migrations := filepath.Join(dir, "..", "..", "..", "migrations")
	if err := goose.Up(db, migrations); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	log.Println("Migrations applied successfully!")
	return db
}

//Пример работы:
//_, err := db.Exec(`
//    INSERT INTO channels (id, title, type, subscribers_counter)
//    VALUES ($1, $2, $3, $4)
//`, "tg_channel_123", "Технологии и код", "p", 4820)
