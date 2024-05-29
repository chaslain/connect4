package main

import (
	"log"
	"slices"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Empty(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	request := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: update.CallbackQuery.InlineMessageID,
			ReplyMarkup:     nil,
		},
		Text: "(empty)",
	}

	bot.Request(request)
}

func getGameText(host string, guest string) string {
	return host + " (🔴) vs " + guest + "(🔵)"
}

func PlayKickQuit(bot *tgbotapi.BotAPI, update *tgbotapi.Update, host string, guest string) {
	buttons := make([]tgbotapi.InlineKeyboardButton, 2)

	buttons[0] = tgbotapi.InlineKeyboardButton{
		Text:         "Kick",
		CallbackData: &KICK_CODE,
	}

	buttons[1] = tgbotapi.InlineKeyboardButton{
		Text:         "Quit",
		CallbackData: &QUIT_CODE,
	}

	allButtons := [][]tgbotapi.InlineKeyboardButton{}

	allButtons = append(allButtons, buttons)
	allButtons = append(allButtons, []tgbotapi.InlineKeyboardButton{{
		Text:         "Play",
		CallbackData: &PLAY_CODE,
	}})

	conf := tgbotapi.EditMessageTextConfig{
		Text: getGameText(host, guest),
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
func rawGameBoard(update *tgbotapi.Update, board Board) tgbotapi.EditMessageTextConfig {
	request := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: update.CallbackQuery.InlineMessageID,
		},
	}

	buttons := make([][]tgbotapi.InlineKeyboardButton, 6)

	for i := 0; i < 6; i++ {
		column := make([]tgbotapi.InlineKeyboardButton, 7)
		for j := 0; j < 7; j++ {
			text := " "

			if len(board.Columns[j].Rows) > i {
				if board.Columns[j].Rows[i] == 1 {
					text = "🔵"
				} else {
					text = "🔴"
				}
			}
			data := strconv.Itoa(j)
			column[j] = tgbotapi.InlineKeyboardButton{
				Text:         text,
				CallbackData: &data,
			}
		}
		buttons[i] = column
	}

	slices.Reverse(buttons)
	request.BaseEdit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	return request
}

func GameBoard(update *tgbotapi.Update, board Board, host string, guest string) {
	request := rawGameBoard(update, board)
	request.Text = getGameText(host, guest)

	resp, e := botapi.Request(request)

	if e != nil {
		log.Default().Println(e.Error())
	}

	if !resp.Ok {
		log.Default().Println(strconv.Itoa(resp.ErrorCode) + " " + resp.Description)
	}
}

func FinishGame(update *tgbotapi.Update, board Board, winner string, host string, guest string) {
	request := rawGameBoard(update, board)
	request.Text = getGameText(host, guest) + "\n" + winner + " Wins!"
	botapi.Request(request)
}

func NewGameMessageCallback(update *tgbotapi.Update, host string) {
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

	request := tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: update.CallbackQuery.InlineMessageID,
			ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: allButtons,
			},
		},
		Text: host + " wants to play connect whore",
	}

	resp, e := botapi.Request(request)
	if e != nil {
		log.Default().Println(e.Error())
	}

	if !resp.Ok {
		log.Default().Println(strconv.Itoa(resp.ErrorCode) + " " + resp.Description)
	}

}

func NewGameMessage(queryId string, username string) {

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

func SendInvalid(update *tgbotapi.Update, message string) {
	config := tgbotapi.NewCallbackWithAlert(update.CallbackQuery.ID, message)
	botapi.Request(config)
}
