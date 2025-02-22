package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgmux "github.com/te5se/tg-mux"
)

const (
	User1ID int64 = 1231532149716294857
)

func TestStart(t *testing.T) {
	bot := tgmux.NewBotAPIMock() // replace on mock for interface + send message

	userStore := NewUserStore()

	router, err := tgmux.NewTGRouter(bot, func(ctx *tgmux.TGContext) (string, error) {
		user, ok := userStore.Get(fmt.Sprint(ctx.Message.Chat.ID))
		if !ok {
			return "", nil
		}

		return user.State, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	startHandler := NewStartHandler(userStore)
	startHandler.Register(router)

	go router.Run(context.Background())

	bot.PushUpdate(tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{
				UserName: "user",
			},
			Chat: &tgbotapi.Chat{
				ID: User1ID,
			},
			Text: "asdf",
		},
	})

	msg := <-bot.GetRecordedChan()

	if msg.Text != "To use the bot, enter /start" {
		t.Errorf("invalid message on wrong input")
	}

}
