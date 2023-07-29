package telegram

import (
	"log"
	"robot/parametrs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func SendMessage(msg string) {
	bot, err := tgbotapi.NewBotAPI(parametrs.TokenBot)
	if err != nil {
		log.Fatal(err)
	}

	message := tgbotapi.NewMessage(parametrs.TgId, msg)
	_, err = bot.Send(message)
	if err != nil {
		log.Fatal(err)
	}
}
