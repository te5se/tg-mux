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
	bot := tgmux.NewBotAPIMock()

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

	bot.PushUpdate(newTextMessage("asdf"))

	msg := <-bot.GetRecordedChan()

	if msg.Text != "To use the bot, enter /start" {
		t.Errorf("invalid message on wrong input")
	}

	upd := newCommandMessage("/start")
	// If you catch an error, you might wanna use these commands to make sure, that your mock message has everything it needs to
	upd.Message.IsCommand()
	upd.Message.Command()

	bot.PushUpdate(upd)

	msg = <-bot.GetRecordedChan()
	if msg.Text != "Hey, enter your username below so I know what to call you 😁" {
		t.Errorf("wrong greeting message %v", msg)
	}

	bot.PushUpdate(newTextMessage(""))

	msg = <-bot.GetRecordedChan()
	if msg.Text != "Please do enter valid username" {
		t.Errorf("wrong 'invlaid username' message")
	}

	bot.PushUpdate(newTextMessage("te5se"))

	msg = <-bot.GetRecordedChan()
	if msg.Text != "Cheers, te5se. You now have access to all the other commands. To see them use the /help command" {
		t.Errorf("wrong 'greeting' message")
	}
}

func TestHelp(t *testing.T) {
	bot := tgmux.NewBotAPIMock()

	userStore := NewUserStore()
	userStore.Store(&User{
		TGID:     fmt.Sprint(User1ID),
		State:    State_None,
		Username: "te5se",
	})

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

	noneHandler := NewNoneHandler()
	noneHandler.Register(router)

	helpHandler := NewHelpHandler()
	helpHandler.Register(router)

	go router.Run(context.Background())

	bot.PushUpdate(newTextMessage("asdf"))

	msg := <-bot.GetRecordedChan()

	if msg.Text != HelpPrompt {
		t.Errorf("invalid message on wrong input")
	}

	bot.PushUpdate(newCommandMessage("/help"))
	msg = <-bot.GetRecordedChan()

	if msg.Text != HelpMessage {
		t.Errorf("invalid help message")
	}
}

func newCommandMessage(command string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{
				UserName: "user",
			},
			Chat: &tgbotapi.Chat{
				ID: User1ID,
			},
			Text: command,
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Length: len(command), Offset: 0},
			},
		},
	}
}

func newTextMessage(text string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{
				UserName: "user",
			},
			Chat: &tgbotapi.Chat{
				ID: User1ID,
			},
			Text: text,
		},
	}
}
