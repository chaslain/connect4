package main

import (
	"connect4/tg/models"
	"log"
	"slices"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type PostEditMessageTextJSONBody struct {
	models.PostEditMessageTextJSONBody
	Method string `json:"method"`
}

type PostAnswerCallbackQueryJSONBody struct {
	models.PostAnswerCallbackQueryJSONBody
	Method string `json:"method"`
}

func Empty(bot *tgbotapi.BotAPI, update *tgbotapi.Update) PostEditMessageTextJSONBody {
	return PostEditMessageTextJSONBody{
		PostEditMessageTextJSONBody: models.PostEditMessageTextJSONBody{
			InlineMessageId: &update.CallbackQuery.InlineMessageID,
			ReplyMarkup:     nil,
			Text:            "(empty)",
		},
		Method: "editMessageText",
	}
}

func getGameText(host string, guest string) string {
	return host + " (🔵) vs " + guest + "(🔴)"
}

func PlayKickQuit(bot *tgbotapi.BotAPI, update *tgbotapi.Update, host string, guest string) PostEditMessageTextJSONBody {
	buttons := make([]models.InlineKeyboardButton, 2)

	buttons[0] = models.InlineKeyboardButton{
		Text:         "Kick",
		CallbackData: &KICK_CODE,
	}

	buttons[1] = models.InlineKeyboardButton{
		Text:         "Quit",
		CallbackData: &QUIT_CODE,
	}

	allButtons := [][]models.InlineKeyboardButton{}

	allButtons = append(allButtons, buttons)
	allButtons = append(allButtons, []models.InlineKeyboardButton{{
		Text:         "Play",
		CallbackData: &PLAY_CODE,
	}})

	return PostEditMessageTextJSONBody{
		PostEditMessageTextJSONBody: models.PostEditMessageTextJSONBody{
			Text:            getGameText(host, guest),
			InlineMessageId: &update.CallbackQuery.InlineMessageID,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: allButtons,
			},
		},
		Method: "editMessageText",
	}
}
func rawGameBoard(update *tgbotapi.Update, board Board) PostEditMessageTextJSONBody {
	request := PostEditMessageTextJSONBody{
		PostEditMessageTextJSONBody: models.PostEditMessageTextJSONBody{
			InlineMessageId: &update.CallbackQuery.InlineMessageID,
		},
		Method: "editMessageText",
	}

	buttons := make([][]models.InlineKeyboardButton, 7)

	for i := 1; i < 7; i++ {
		column := make([]models.InlineKeyboardButton, 7)
		for j := 0; j < 7; j++ {
			text := " "

			if len(board.Columns[j].Rows) > i-1 {
				if board.Columns[j].Rows[i-1] == 1 {
					text = "🔵"
				} else {
					text = "🔴"
				}
			}
			data := strconv.Itoa(j)
			column[j] = models.InlineKeyboardButton{
				Text:         text,
				CallbackData: &data,
			}
		}
		buttons[i] = column
	}
	buttons[0] = []models.InlineKeyboardButton{
		{
			Text:         "Claim Win",
			CallbackData: &CLAIM_CODE,
		},
		{
			Text:         "Resign",
			CallbackData: &RESIGN_CODE,
		},
	}

	slices.Reverse(buttons)
	request.ReplyMarkup = &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	return request
}

func GetGameBoard(update *tgbotapi.Update, board Board, host string, guest string, moveNum int) PostEditMessageTextJSONBody {
	request := rawGameBoard(update, board)
	playerMove := "🔴 " + guest + " to move"
	if moveNum%2 == 1 {
		playerMove = "🔵 " + host + " to move"
	}

	request.Text = getGameText(host, guest) + "\n" + playerMove
	return request
}

func FinishDrawnGame(update *tgbotapi.Update, board Board, host string, guest string) PostEditMessageTextJSONBody {
	request := rawGameBoard(update, board)
	request.Text = getGameText(host, guest) + "\n" + "Drawn"
	return request
}

func FinishGame(update *tgbotapi.Update, board Board, winner string, host string, guest string) PostEditMessageTextJSONBody {
	request := rawGameBoard(update, board)
	request.Text = getGameText(host, guest) + "\n" + winner + " Wins!"
	return request
}

func NewGameMessageCallback(update *tgbotapi.Update, host string) PostEditMessageTextJSONBody {
	buttons := make([]models.InlineKeyboardButton, 2)
	buttons[0] = models.InlineKeyboardButton{
		Text:         "Join",
		CallbackData: &JOIN_CODE,
	}

	buttons[1] = models.InlineKeyboardButton{
		Text:         "Quit",
		CallbackData: &QUIT_CODE,
	}

	allButtons := [][]models.InlineKeyboardButton{}

	allButtons = append(allButtons, buttons)

	request := PostEditMessageTextJSONBody{
		PostEditMessageTextJSONBody: models.PostEditMessageTextJSONBody{
			InlineMessageId: &update.CallbackQuery.InlineMessageID,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: allButtons,
			},

			Text: host + " wants to play connect 4",
		},
		Method: "editMessageText",
	}

	return request
}

func InlineQueryMessage(queryId string, username string, ranking int, elo int, total int, top10 string) {

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
			Text: username + " wants to play connect 4",
		},
		ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: allButtons,
		},
	})

	if ranking != -1 {
		results = append(results, tgbotapi.InlineQueryResultArticle{
			ID:          "checkRank",
			Type:        "Article",
			Title:       "Rating",
			Description: "My current connect 4 rating is...",
			InputMessageContent: tgbotapi.InputTextMessageContent{
				Text: "My current connect4 rating: " + strconv.Itoa(elo) + "\n " + strconv.Itoa(ranking) + " of " + strconv.Itoa(total) + " players",
			},
		})
	}

	if len(top10) > 0 {
		results = append(results, tgbotapi.InlineQueryResultArticle{
			ID:          "leaderBoard",
			Type:        "Article",
			Title:       "Top 10 players",
			Description: "Top 10 players",
			InputMessageContent: tgbotapi.InputTextMessageContent{
				Text: top10,
			},
		})
	}

	ic := tgbotapi.InlineConfig{
		InlineQueryID: queryId,
		Results:       results,
		IsPersonal:    true,
	}

	result, err := botapi.Request(ic)
	if err != nil {
		log.Default().Println("Failed to call tg API - " + err.Error())
	} else if !result.Ok {
		log.Default().Println("Error response from tg API " + strconv.Itoa(result.ErrorCode) + " " + result.Description)
	}
}

func SendInvalid(update *tgbotapi.Update, message string) PostAnswerCallbackQueryJSONBody {
	t := true
	return PostAnswerCallbackQueryJSONBody{
		PostAnswerCallbackQueryJSONBody: models.PostAnswerCallbackQueryJSONBody{
			ShowAlert:       &t,
			Text:            &message,
			CallbackQueryId: update.CallbackQuery.ID,
		},
		Method: "answerCallbackQuery",
	}
}
