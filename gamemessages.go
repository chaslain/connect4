package main

import (
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func PlayKick(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	buttons := make([]tgbotapi.InlineKeyboardButton, 2)
	buttons[0] = tgbotapi.InlineKeyboardButton{
		Text:         "Play",
		CallbackData: &JOIN_CODE,
	}

	buttons[1] = tgbotapi.InlineKeyboardButton{
		Text:         "Kick",
		CallbackData: &QUIT_CODE,
	}

	allButtons := [][]tgbotapi.InlineKeyboardButton{}

	allButtons = append(allButtons, buttons)

	conf := tgbotapi.EditMessageTextConfig{
		Text: update.CallbackQuery.From.FirstName + " has joined.",
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: update.CallbackQuery.InlineMessageID,
			ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: allButtons,
			},
		},
	}

	_, e := bot.Request(conf)
	if e != nil {
		log.Default().Println(e.Error())
	}
}

func JoinQuit(queryId string, username string) {

	var results []interface{}

	buttons := make([]tgbotapi.InlineKeyboardButton, 2)
	buttons[0] = tgbotapi.InlineKeyboardButton{
		Text:         "Join",
		CallbackData: &JOIN_CODE,
	}

	buttons[1] = tgbotapi.InlineKeyboardButton{
		Text:         "Quit",
		CallbackData: &QUIT_CODE,
	}

	allButtons := [][]tgbotapi.InlineKeyboardButton{}

	allButtons = append(allButtons, buttons)

	results = append(results, tgbotapi.InlineQueryResultArticle{
		ID:          "connect4",
		Type:        "Article",
		Title:       "Connect 4",
		Description: "Play connect 4!",
		InputMessageContent: tgbotapi.InputTextMessageContent{
			Text: username + " wants to play connect whore",
		},
		ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: allButtons,
		},
	})

	ic := tgbotapi.InlineConfig{
		InlineQueryID: queryId,
		Results:       results,
	}

	result, err := botapi.Request(ic)
	if err != nil {
		log.Default().Println("Failed to call tg API - " + err.Error())
	} else if !result.Ok {
		log.Default().Println("Error response from tg API " + strconv.Itoa(result.ErrorCode) + " " + result.Description)
	}
}
