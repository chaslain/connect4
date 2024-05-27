package main

import (
	"database/sql"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

func InitDb(dsn string) (*sql.DB, error) {
	result, err := sql.Open("sqlite3", dsn)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func CreateUser(db *sql.DB, tgUserId int64, name string) {
	date := time.Now()
	db.Exec("INSERT INTO user (tg_id, date_created, elo, first_name) VALUES (?, ?, 1500, ?)", tgUserId, date, name)
}

func CreateGame(db *sql.DB, update tgbotapi.Update) {
	date := time.Now()
	query := "INSERT OR REPLACE INTO game (one_user_tg_id, date_created, hosted_message_id, move_number) VALUES (?, ?, ?, ?)"
	db.Exec(query, update.ChosenInlineResult.From.ID, date, update.ChosenInlineResult.InlineMessageID, 0)
}

func JoinGame(db *sql.DB, update tgbotapi.Update) {
	query := "UPDATE game SET two_user_tg_id = ? WHERE hosted_message_id = ?"
	db.Exec(query, update.CallbackQuery.From.ID, update.CallbackQuery.InlineMessageID)
}

func GetHost(db *sql.DB, InlineMessageID string) string {
	q := `SELECT a.first_name
		    FROM user a
			JOIN game b ON (a.tg_id = b.one_user_tg_id)
			WHERE b.hosted_message_id = ?
	`

	r := db.QueryRow(q, InlineMessageID)

	var result string
	e := r.Scan(&result)
	if e == nil {
		log.Default().Println("Host found: " + result)
		return result
	} else {
		log.Default().Println(e.Error())
	}
	return ""
}

func LeaveGame(db *sql.DB, update tgbotapi.Update) bool {
	query := "SELECT one_user_tg_id, two_user_tg_id FROM game WHERE hosted_message_id = ?"
	row := db.QueryRow(query, update.CallbackQuery.InlineMessageID)
	if row == nil {
		return false
	}
	var one int64
	var two int64

	row.Scan(&one, &two)

	var updatequery string
	result := false
	if update.CallbackQuery.From.ID == one {
		updatequery = "UPDATE game SET one_user_tg_id = NULL WHERE hosted_message_id = ?"
		result = true
	} else if update.CallbackQuery.From.ID == two {
		updatequery = "UPDATE game SET two_user_tg_id = NULL WHERE hosted_message_id = ?"
	}
	db.Exec(updatequery, update.CallbackQuery.InlineMessageID)
	return result
}

func UpdateState(db *sql.DB, gameId string, serialized string) {
	query := "UPDATE game SET game_board = ?, move_number = move_number+1 WHERE hosted_message_id = ?"
	db.Exec(query, serialized, gameId)
}

func ReadGame(db *sql.DB, gameId string) (int64, int64, string, int) {
	query := "SELECT game_board, one_user_tg_id, two_user_tg_id, move_number FROM game WHERE hosted_message_id = ?"
	var game string
	var host int64
	var guest int64
	var move_number int

	data := db.QueryRow(query, gameId)
	data.Scan(&game, &host, &guest, &move_number)
	return host, guest, game, move_number
}
