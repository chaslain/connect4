package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigyaml"
	"github.com/gin-gonic/gin"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type config struct {
	TelegramBotToken string
	SchemaDsn        string
	ImageUrl         string
	Debug            bool `default:"false"`
	BaseElo          int
	EloK             float32
	Port             string
	PublicKeyPath    string
	SslPort          string
	WebhookURL       string
	KillAge          int
}

var env config

var botapi *tg.BotAPI

var db *sql.DB

var m *sync.Mutex

// not consts as they are passed as references downstream.
// effectively consts.
var QUIT_CODE string = "q"
var JOIN_CODE string = "j"
var PLAY_CODE string = "p"
var KICK_CODE string = "k"
var CLAIM_CODE string = "c"
var RESIGN_CODE string = "r"

func main() {
	m = new(sync.Mutex)
	initConfig()
	var dbError error
	db, dbError = InitDb(env.SchemaDsn)
	if dbError != nil {
		panic("Failed to init sqlite - " + dbError.Error())
	}
	informWebhook(env.WebhookURL, env.PublicKeyPath)
	r := gin.Default()
	r.POST("/", listener)
	r.Run()
}

func informWebhook(url string, publickey string) {
	var err error
	botapi, err = tg.NewBotAPI(env.TelegramBotToken)
	if err != nil {
		panic("Failed to authorize tg - " + err.Error())
	}

	keyfile, err := os.Open(publickey)

	if err != nil {
		panic("Failed to read key file")
	}
	wh, webhookerr := tg.NewWebhookWithCert(url, tg.FileReader{
		Reader: keyfile,
	})

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
	m.Lock()
	defer m.Unlock()
	if env.Debug {
		chars, _ := io.ReadAll(context.Request.Body)
		log.Default().Println(string(chars))
		context.Request.Body = io.NopCloser(bytes.NewReader(chars))
	}
	update := tg.Update{}
	context.BindJSON(&update)
	log.Default().Print("Processing update " + strconv.Itoa(update.UpdateID))

	if update.InlineQuery != nil {
		ranking, elo := QueryPlayerRanking(db, update.InlineQuery.From.ID)
		total := QueryTotalPlayerCount(db)
		InlineQueryMessage(update.InlineQuery.ID, update.SentFrom().FirstName, ranking, elo, total, Top10LeaderBoard(db))
		context.Status(204)
	} else if update.CallbackQuery != nil {
		response := handleInput(&update)
		if response != nil {
			context.JSON(200, response)
			if env.Debug {
				j, _ := json.Marshal(response)
				log.Default().Println(string(j))
			}
		}
		return
	} else if update.ChosenInlineResult != nil {
		if update.ChosenInlineResult.ResultID == "connect4" {
			CreateUser(db, update.ChosenInlineResult.From.ID, update.ChosenInlineResult.From.FirstName, env.BaseElo)
			CreateGame(db, update)
		}
		context.Status(204)
	}
}

func handleInput(update *tg.Update) interface{} {
	if update.CallbackQuery.Data == JOIN_CODE {
		CreateUser(db, update.CallbackQuery.From.ID, update.CallbackQuery.From.FirstName, env.BaseElo)
		JoinGame(db, *update)
		host, _ := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
		guest := update.CallbackQuery.From.FirstName
		if host == "" || guest == "" {
			Empty(botapi, update)
		}
		a, b := QueryElo(db, update.CallbackQuery.InlineMessageID)
		return PlayKickQuit(botapi, update, host+" "+parenthesizeInt(a), guest+" "+parenthesizeInt(b))
	} else if update.CallbackQuery.Data == QUIT_CODE {
		host, _ := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
		hostLeft, valid := LeaveGame(db, *update)
		if !valid {
			return SendInvalid(update, "Nah fam")
		}
		if hostLeft {
			return Empty(botapi, update)
		} else {
			return NewGameMessageCallback(update, host)
		}
	} else if update.CallbackQuery.Data == PLAY_CODE {
		hostId := GetHostId(db, update.CallbackQuery.InlineMessageID)
		if hostId != update.CallbackQuery.From.ID {
			return SendInvalid(update, "Nah fam")
		}
		guestId := GetGuestId(db, update.CallbackQuery.InlineMessageID)
		if guestId == 0 {
			return SendInvalid(update, "They left as soon as you hit join. Sry")
		}
		board := EmptyBoard()
		UpdateState(db, update.CallbackQuery.InlineMessageID, GetSerial(board), 0)
		hostName, guestName := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
		a, b := QueryElo(db, update.CallbackQuery.InlineMessageID)
		return GetGameBoard(update, board, hostName+" "+parenthesizeInt(a), guestName+" "+parenthesizeInt(b), 0)
	} else if update.CallbackQuery.Data == KICK_CODE {
		hostId := GetHostId(db, update.CallbackQuery.InlineMessageID)
		if hostId != update.CallbackQuery.From.ID {
			return SendInvalid(update, "Nah fam")
		}

		return NewGameMessageCallback(update, update.CallbackQuery.From.FirstName)
	} else if update.CallbackQuery.Data == CLAIM_CODE {
		host, guest, game, move_number := ReadGame(db, update.CallbackQuery.InlineMessageID)
		hostToMove := move_number%2 == 0

		if update.CallbackQuery.From.ID != host && update.CallbackQuery.From.ID != guest {
			return SendInvalid(update, "Nah fam")
		}

		if !hostToMove && update.CallbackQuery.From.ID == guest {
			return SendInvalid(update, "Nah fam")
		} else if hostToMove && update.CallbackQuery.From.ID == host {
			return SendInvalid(update, "Nah fam")
		}

		moveTime := ReadGameLastMove(db, update.CallbackQuery.InlineMessageID)

		if moveTime+(int64(env.KillAge)*60) < time.Now().Unix() {

			hostName, guestName := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
			winnerName := hostName
			winner := 1
			if hostToMove {
				winner = -1
				winnerName = guestName
			}

			olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)
			a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, game, winner, env.EloK, env.BaseElo)
			hostName = getFinishData(hostName, olda, a)
			guestName = getFinishData(guestName, oldb, b)
			return FinishGame(update, GetGame(game), winnerName, hostName, guestName)
		} else {
			return SendInvalid(update, "You must wait "+strconv.Itoa(env.KillAge)+" minutes to claim your win.")
		}

	} else if update.CallbackQuery.Data == RESIGN_CODE {
		host, guest, game, _ := ReadGame(db, update.CallbackQuery.InlineMessageID)

		if update.CallbackQuery.From.ID != host && update.CallbackQuery.From.ID != guest {
			return SendInvalid(update, "Nah fam")
		}

		hostName, guestName := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)
		olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)

		if update.CallbackQuery.From.ID == host {
			a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, game, -1, env.EloK, env.BaseElo)
			hostName = getFinishData(hostName, olda, a)
			guestName = getFinishData(guestName, oldb, b)
			return FinishGame(update, GetGame(game), guestName, hostName, guestName)
		} else {
			a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, game, 1, env.EloK, env.BaseElo)
			hostName = getFinishData(hostName, olda, a)
			guestName = getFinishData(guestName, oldb, b)
			return FinishGame(update, GetGame(game), hostName, hostName, guestName)
		}
	} else {
		host, guest, game, move_number := ReadGame(db, update.CallbackQuery.InlineMessageID)
		hostMove := move_number%2 == 0

		if update.CallbackQuery.From.ID != host && hostMove {
			return SendInvalid(update, "It is not your turn!")
		} else if !hostMove && update.CallbackQuery.From.ID != guest {
			return SendInvalid(update, "It is not your turn!")
		}

		board := GetGame(game)
		column, _ := strconv.Atoi(update.CallbackQuery.Data)

		hostName, guestName := GetPlayerNames(db, update.CallbackQuery.InlineMessageID)

		if hostMove {
			if !PlayMove(&board, column, 1) {
				SendInvalid(update, "Invalid move!")
				return nil
			}
			if CheckForWin(&board, column, 1) {
				olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)
				a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, GetSerial(board), 1, env.EloK, env.BaseElo)
				hostName = getFinishData(hostName, olda, a)
				guestName = getFinishData(guestName, oldb, b)
				return FinishGame(update, board, update.CallbackQuery.From.FirstName, hostName, guestName)
			}
		} else {
			if !PlayMove(&board, column, 2) {
				return SendInvalid(update, "Invalid move!")
			}
			if CheckForWin(&board, column, 2) {
				olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)
				a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, GetSerial(board), -1, env.EloK, env.BaseElo)
				hostName = getFinishData(hostName, olda, a)
				guestName = getFinishData(guestName, oldb, b)
				return FinishGame(update, board, update.CallbackQuery.From.FirstName, hostName, guestName)
			} else {
				if move_number == 41 {
					olda, oldb := QueryElo(db, update.CallbackQuery.InlineMessageID)
					a, b := CloseGame(db, update.CallbackQuery.InlineMessageID, GetSerial(board), 0, env.EloK, env.BaseElo)
					hostName = getFinishData(hostName, olda, a)
					guestName = getFinishData(guestName, oldb, b)
					return FinishDrawnGame(update, board, hostName, guestName)
				}
			}
		}

		game = GetSerial(board)

		a, b := QueryElo(db, update.CallbackQuery.InlineMessageID)
		UpdateState(db, update.CallbackQuery.InlineMessageID, game, move_number+1)
		result := GetGameBoard(update, board, hostName+" "+parenthesizeInt(a), guestName+" "+parenthesizeInt(b), move_number)
		return &result
	}

	return nil
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

func getFinishData(hostName string, elo int, change int) string {
	s := strconv.Itoa(int(change))
	if change >= 0 {
		s = "+" + strconv.Itoa(int(change))
	}
	return hostName + " " + parenthesize(strconv.Itoa(elo)+s)
}
