package main

import (
	"database/sql"
	"log"
	"strconv"
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

func CreateUser(db *sql.DB, tgUserId int64, name string, offset int) {
	date := time.Now()
	db.Exec("INSERT INTO user (tg_id, date_created, elo, first_name) VALUES (?, ?, ?, ?)", tgUserId, date, offset, name)
}

func CreateGame(db *sql.DB, update tgbotapi.Update) {
	date := time.Now()
	query := "INSERT OR REPLACE INTO game (one_user_tg_id, date_created, hosted_message_id, move_number) VALUES (?, ?, ?, ?)"
	_, err := db.Exec(query, update.ChosenInlineResult.From.ID, date, update.ChosenInlineResult.InlineMessageID, 0)
	if err != nil {
		log.Default().Println(err.Error())
	}
}

func JoinGame(db *sql.DB, update tgbotapi.Update) {
	query := "UPDATE game SET two_user_tg_id = ? WHERE hosted_message_id = ?"
	db.Exec(query, update.CallbackQuery.From.ID, update.CallbackQuery.InlineMessageID)
}

func GetHostId(db *sql.DB, InlineMessageID string) int64 {
	q := `SELECT a.tg_id
		    FROM user a
			JOIN game b ON (a.tg_id = b.one_user_tg_id)
			WHERE b.hosted_message_id = ?
	`

	r := db.QueryRow(q, InlineMessageID)

	var result int64
	e := r.Scan(&result)
	if e != nil {
		log.Default().Println(e.Error())
	}
	return result
}

func GetGuestId(db *sql.DB, InlineMessageID string) int64 {
	q := `SELECT a.tg_id
	FROM user a
	JOIN game b ON (a.tg_id = b.two_user_tg_id)
	WHERE b.hosted_message_id = ?
`

	r := db.QueryRow(q, InlineMessageID)

	var result int64
	e := r.Scan(&result)
	if e != nil {
		log.Default().Println(e.Error())
	}
	return result
}

func GetPlayerNames(db *sql.DB, InlineMessageID string) (string, string) {
	q := `SELECT a.first_name, c.first_name
		    FROM user a
			JOIN game b ON (a.tg_id = b.one_user_tg_id)
			JOIN user c ON (b.two_user_tg_id = c.tg_id)
			WHERE b.hosted_message_id = ?
	`

	r := db.QueryRow(q, InlineMessageID)

	var result, result2 string
	e := r.Scan(&result, &result2)
	if e == nil {
		log.Default().Println("Host found: " + result)
		return result, result2
	} else {
		log.Default().Println(e.Error())
	}
	return "", ""
}

// returns two booleans -
// first is true if the host left the game, false if guest.
// the other is true if the leave request was valid at all. No one leaves the game if it is invalid.
func LeaveGame(db *sql.DB, update tgbotapi.Update) (bool, bool) {
	query := "SELECT one_user_tg_id, two_user_tg_id FROM game WHERE hosted_message_id = ?"
	row := db.QueryRow(query, update.CallbackQuery.InlineMessageID)
	if row == nil {
		return false, false
	}
	var one int64
	var two int64

	row.Scan(&one, &two)

	var updatequery string
	if update.CallbackQuery.From.ID == one {
		updatequery = "UPDATE game SET one_user_tg_id = NULL WHERE hosted_message_id = ?"
		db.Exec(updatequery, update.CallbackQuery.InlineMessageID)
		return true, true
	} else if update.CallbackQuery.From.ID == two {
		updatequery = "UPDATE game SET two_user_tg_id = NULL WHERE hosted_message_id = ?"
		db.Exec(updatequery, update.CallbackQuery.InlineMessageID)
		return false, true
	} else {
		return false, false
	}
}

func UpdateState(db *sql.DB, gameId string, serialized string) {
	query := "UPDATE game SET game_board = ?, move_number = move_number+1, last_move = ? WHERE hosted_message_id = ?"
	db.Exec(query, serialized, time.Now().UTC(), gameId)
}

// if the host wins, pass 1 for winner. else, -1. 0 for draw
func CloseGame(db *sql.DB, gameId string, serialized string, winner int, k float32, offset int) (int, int) {
	UpdateState(db, gameId, serialized)
	return handleElo(db, gameId, winner, k, offset)
}

func handleElo(db *sql.DB, gameId string, winner int, k float32, offset int) (int, int) {

	a, b := QueryElo(db, gameId)
	playerone, playertwo := GetELoAdjustments(a, b, winner, k)

	hostId := GetHostId(db, gameId)
	guestId := GetGuestId(db, gameId)

	// games against self
	if hostId == guestId {
		return 0, 0
	}

	query := `
		INSERT INTO game_outcome (hosted_message_id, tg_id, elo_adjustment)
		VALUES (?, ?, ?)
	`
	db.Exec(query, gameId, hostId, playerone)
	db.Exec(query, gameId, guestId, playertwo)
	UpdateElo(db, hostId, offset)
	UpdateElo(db, guestId, offset)
	return playerone, playertwo
}

func ReadGameLastMove(db *sql.DB, gameId string) int64 {
	query := "SELECT last_move FROM game WHERE hosted_message_id = ?"

	qRow := db.QueryRow(query, gameId)
	var result time.Time
	e := qRow.Scan(&result)
	if e != nil {
		panic(e.Error())
	}
	return result.Unix()
}

func ReadGame(db *sql.DB, gameId string) (int64, int64, string, int) {
	query := `SELECT game_board, one_user_tg_id, two_user_tg_id, move_number 
			    FROM game a
	       LEFT JOIN game_outcome b ON (a.hosted_message_id = b.hosted_message_id)
			   WHERE a.hosted_message_id = ? AND b.hosted_message_id IS NULL`
	var game string
	var host int64
	var guest int64
	var move_number int

	data := db.QueryRow(query, gameId)
	data.Scan(&game, &host, &guest, &move_number)
	return host, guest, game, move_number
}

// get total players
func QueryTotalPlayerCount(db *sql.DB) int {
	query := "SELECT COUNT(*) FROM user"
	var result int
	row := db.QueryRow(query)
	row.Scan(&result)
	return result
}

// returns ranking number, elo
func QueryPlayerRanking(db *sql.DB, tg_id int64) (int, int) {
	query := `
		SELECT rank, elo FROM (
		SELECT RANK() OVER (ORDER BY elo DESC) rank, elo, tg_id
		FROM user
		) x
		WHERE tg_id = ?
	`

	row := db.QueryRow(query, tg_id)

	var ranking, elo int
	row.Scan(&ranking, &elo)
	if row.Err() != nil {
		log.Default().Println("Error getting player ranking: " + row.Err().Error())
		return -1, -1
	}
	return ranking, elo
}

func QueryPlayerElo(db *sql.DB, tg_id int64, offset int) int {
	query := `
		SELECT elo FROM user WHERE tg_id = ?
	`

	var result int
	row := db.QueryRow(query, tg_id)
	row.Scan(&result)
	if result == 0 {
		return offset
	}
	return result
}

func QueryElo(db *sql.DB, game_id string) (int, int) {
	query := `
		SELECT b.elo, c.elo
		FROM game a
		JOIN user b ON (a.one_user_tg_id = b.tg_id)
		LEFT JOIN user c ON (a.two_user_tg_id = c.tg_id)
		WHERE a.hosted_message_id = ?
	`

	var result, result2 int
	row := db.QueryRow(query, game_id)
	row.Scan(&result, &result2)
	return result, result2
}

func UpdateElo(db *sql.DB, tg_user_id int64, offset int) {
	query := `
		UPDATE user SET elo = 
		(SELECT SUM(elo_adjustment)
			  FROM game_outcome
			 WHERE tg_id = ?) + ?
		WHERE tg_id = ?
	`

	db.Exec(query, tg_user_id, offset, tg_user_id)

}

func Top10LeaderBoard(db *sql.DB) string {
	sql := `
		SELECT first_name, elo FROM user ORDER BY elo DESC LIMIT 10
	`

	data, err := db.Query(sql)
	if err != nil {
		log.Default().Println("Error querying top 10: " + err.Error())
		return "(Could not get top 10)"
	}

	result := ""

	for i := 0; i < 10; i++ {
		var elo int
		var name string
		if !data.Next() {
			break
		}
		data.Scan(&name, &elo)
		result += strconv.Itoa(i+1) + " " + name + ": " + strconv.Itoa(elo) + "\n"
	}

	data.Close()
	return result
}
