package main

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"strconv"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
	"github.com/gin-gonic/gin"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type config struct {
	TelegramBotToken string
	SchemaDsn        string
	WebhookUrl       string
	ImageUrl         string
	Debug            bool `default:"false"`
	BaseElo          int
	EloK             float32
}

var env config

var botapi *tg.BotAPI

var db *sql.DB

// not consts as they are passed as references downstream.
// effectively consts.
var QUIT_CODE string = "q"
var JOIN_CODE string = "j"
var PLAY_CODE string = "p"
var KICK_CODE string = "k"

func main() {
	initConfig()
	var dbError error
	db, dbError = InitDb(env.SchemaDsn)
	if dbError != nil {
		panic("Failed to init sqlite - " + dbError.Error())
	}
	informWebhook()
	r := gin.Default()
	r.POST("/", listener)
	r.Run()
}

func informWebhook() {
	var err error
	botapi, err = tg.NewBotAPI(env.TelegramBotToken)
	if err != nil {
		panic("Failed to authorize tg - " + err.Error())
	}

	wh, webhookerr := tg.NewWebhook(env.WebhookUrl)
	if webhookerr != nil {
		panic("Failed to initialize webhook - " + webhookerr.Error())
	}
	wh.DropPendingUpdates = true
	wh.AllowedUpdates = append(wh.AllowedUpdates, "inline_query", "callback_query", "chosen_inline_result")

	resp, responseError := botapi.Request(wh)
	if responseError != nil {
		panic("Failed calling tg api: " + responseError.Error())
	}

	if !resp.Ok {
		panic("Bad response from tg: " + strconv.Itoa(resp.ErrorCode) + " - " + resp.Description)
	}
	log.Default().Println("Successfully initialized webhook.")
}

func listener(context *gin.Context) {
	if env.Debug {
		chars, _ := io.ReadAll(context.Request.Body)
		log.Default().Println(string(chars))
		context.Request.Body = io.NopCloser(bytes.NewReader(chars))
	}
	update := tg.Update{}
	context.BindJSON(&update)
	log.Default().Print("Processing update " + strconv.Itoa(update.UpdateID))

	if update.InlineQuery != nil {
		elo := QueryPlayerElo(db, update.InlineQuery.From.ID)
		NewGameMessage(update.InlineQuery.ID, update.SentFrom().FirstName+parenthesizeInt(elo))
	} else if update.CallbackQuery != nil {
		handleInput(&update)
	} else if update.ChosenInlineResult != nil {
		CreateUser(db, update.ChosenInlineResult.From.ID, update.ChosenInlineResult.From.FirstName)
		CreateGame(db, update)
	}
	context.Status(204)
}

func handleInput(update *tg.Update) {
	if update.CallbackQuery.Data == JOIN_CODE {
		CreateUser(db, update.CallbackQuery.From.ID, update.CallbackQuery.From.FirstName)
		JoinGame(db, *update)
		host, guest := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
		a, b := QueryElo(db, update.CallbackQuery.InlineMessageID)
		PlayKickQuit(botapi, update, host+" "+parenthesizeInt(a), guest+" "+parenthesizeInt(b))
	} else if update.CallbackQuery.Data == QUIT_CODE {
		if LeaveGame(db, *update) {
			Empty(botapi, update)
		} else {
			host, _ := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
			NewGameMessageCallback(update, host)
		}
	} else if update.CallbackQuery.Data == PLAY_CODE {
		hostId := GetHostId(db, update.CallbackQuery.InlineMessageID)
		if hostId != update.CallbackQuery.From.ID {
			SendInvalid(update, "Nah fam")
			return
		}
		board := EmptyBoard()
		UpdateState(db, update.CallbackQuery.InlineMessageID, GetSerial(board))
		hostName, guestName := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
		a, b := QueryElo(db, update.CallbackQuery.InlineMessageID)
		GameBoard(update, board, hostName+" "+parenthesizeInt(a), guestName+" "+parenthesizeInt(b))
	} else if update.CallbackQuery.Data == KICK_CODE {
		hostId := GetHostId(db, update.CallbackQuery.InlineMessageID)
		if hostId != update.CallbackQuery.From.ID {
			SendInvalid(update, "Nah fam")
			return
		}

		NewGameMessageCallback(update, update.CallbackQuery.From.FirstName)
	} else {
		host, guest, game, move_number := ReadGame(db, update.CallbackQuery.InlineMessageID)
		hostMove := move_number%2 == 1

		if update.CallbackQuery.From.ID != host && hostMove {
			SendInvalid(update, "It is not your turn!")
			return
		} else if !hostMove && update.CallbackQuery.From.ID != guest {
			SendInvalid(update, "It is not your turn!")
			return
		}

		board := GetGame(game)
		column, _ := strconv.Atoi(update.CallbackQuery.Data)

		hostName, guestName := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)

		if hostMove {
			if !PlayMove(&board, column, 1) {
				SendInvalid(update, "Invalid move!")
				return
			}
			if CheckForWin(&board, column, 1) {
				olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)
				a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, GetSerial(board), -1, env.EloK)
				hostName = getFinishData(hostName, olda, a)
				guestName = getFinishData(guestName, oldb, b)
				FinishGame(update, board, update.CallbackQuery.From.FirstName, hostName, guestName)
				return
			}
		} else {
			if !PlayMove(&board, column, 2) {
				SendInvalid(update, "Invalid move!")
				return
			}
			if CheckForWin(&board, column, 2) {
				olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)
				a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, GetSerial(board), -1, env.EloK)
				hostName = getFinishData(hostName, olda, a)
				guestName = getFinishData(guestName, oldb, b)
				FinishGame(update, board, update.CallbackQuery.From.FirstName, hostName, guestName)
				return
			} else {
				if move_number == 42 {
					CloseGame(db, update.CallbackQuery.InlineMessageID, GetSerial(board), 0, env.EloK)
				}
			}
		}

		game = GetSerial(board)

		a, b := QueryElo(db, update.CallbackQuery.InlineMessageID)
		UpdateState(db, update.CallbackQuery.InlineMessageID, game)
		GameBoard(update, board, hostName+" "+parenthesizeInt(a), guestName+" "+parenthesizeInt(b))
	}
}

func initConfig() {
	env = config{}
	loader := aconfig.LoaderFor(&env, aconfig.Config{
		Files: []string{"resources/config.yaml"},
		FileDecoders: map[string]aconfig.FileDecoder{
			".yaml": aconfigyaml.New(),
		},
	})

	err := loader.Load()
	if err != nil {
		panic("Failed to load configs. " + err.Error())
	}

	log.Default().Print("Successfully loaded configs.")
}

func parenthesize(data string) string {
	return "(" + data + ")"
}

func parenthesizeInt(data int) string {
	return "(" + strconv.Itoa(data) + ")"
}

func getFinishData(hostName string, elo int, change float32) string {
	s := strconv.Itoa(int(change))
	if change >= 0 {
		s = "+" + strconv.Itoa(int(change))
	}
	return hostName + " " + parenthesize(strconv.Itoa(elo)+s)
}
