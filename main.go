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
		NewGameMessage(update.InlineQuery.ID, update.SentFrom().FirstName)
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
		PlayKickQuit(botapi, update, GetHost(db, update.CallbackQuery.InlineMessageID))
	} else if update.CallbackQuery.Data == QUIT_CODE {
		LeaveGame(db, *update)
		log.Default().Println("Quit")
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
