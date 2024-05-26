package main

import (
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
}

var env config

var botapi *tg.BotAPI

func main() {
	initConfig()
	informWebhook()
	r := gin.Default()
	r.POST("/handle", listener)
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
	wh.AllowedUpdates = append(wh.AllowedUpdates, "inline_query", "callback_query")

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
	update := tg.Update{}
	context.BindJSON(&update)
	log.Default().Print("Processing update " + strconv.Itoa(update.UpdateID))
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
