package main

import (
	"fmt"
	"log"
	"os"
	router "tg-mux"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	tgKey, exists := os.LookupEnv("TEST_TG_KEY")
	if !exists {
		log.Fatal("token not found in env TEST_TG_KEY")
	}

	bot, err := tgbotapi.NewBotAPI(tgKey)
	if err != nil {
		log.Fatal(err)
	}

	userStore := NewUserStore()

	router, err := router.NewTGRouter(bot, func(ctx *router.TGContext) (string, error) {
		user, ok := userStore.Get(fmt.Sprint(ctx.Message.Chat.ID))
		if !ok {
			return "", nil
		}

		return user.State, nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
