package main

import (
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgmux "github.com/te5se/tg-mux"
)

type StartHandler struct {
	UserStore *UserStore
}

func NewStartHandler(userStore *UserStore) *StartHandler {
	return &StartHandler{
		UserStore: userStore,
	}
}

func (h *StartHandler) Register(router *tgmux.TGRouter) {
	router.RegisterCommand("start", h.HandleStartCommand)
	router.RegisterStateHandler(State_Start, h.HandleUsernameEnter, h.Cleanup)
}

func (h *StartHandler) HandleStartCommand(ctx *tgmux.TGContext) (tgbotapi.MessageConfig, error) {
	user, ok := h.UserStore.Get(fmt.Sprint(ctx.Message.Chat.ID))
	if ok {
		return tgbotapi.NewMessage(ctx.Message.Chat.ID, fmt.Sprintf("You're already registered, you know. Your username is %v.", user.Username)), nil
	}

	h.UserStore.Store(&User{
		TGID:  fmt.Sprint(ctx.Message.Chat.ID),
		State: State_Start,
	})

	return tgbotapi.NewMessage(ctx.Message.Chat.ID, "Hey, enter your username below so I know what to call you 😁"), nil
}

func (h *StartHandler) HandleUsernameEnter(ctx *tgmux.TGContext) (tgbotapi.MessageConfig, error) {
	user, ok := h.UserStore.Get(fmt.Sprint(ctx.Message.Chat.ID))
	if !ok {
		return tgbotapi.MessageConfig{}, errors.New("not found")
	}

	if ctx.Message.Text == "" {
		return tgbotapi.NewMessage(ctx.Message.Chat.ID, "Please do enter valid username"), nil
	}

	user.Username = ctx.Message.Text
	user.State = State_None

	h.UserStore.Store(user)

	return tgbotapi.NewMessage(ctx.Message.Chat.ID, fmt.Sprintf("Cheers, %v. You now have access to all the other commands. To see them use the /help command", user.Username)), nil
}

func (h *StartHandler) Cleanup(tgCtx *tgmux.TGContext) error {
	h.UserStore.cache.Delete(fmt.Sprint(tgCtx.Message.Chat.ID))

	return nil
}
