package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgmux "github.com/te5se/tg-mux"
)

type WhoamiHandler struct {
	userStore *UserStore
}

func NewWhoamiHandler(userStore *UserStore) *WhoamiHandler {
	return &WhoamiHandler{
		userStore: userStore,
	}
}

func (h *WhoamiHandler) Register(router *tgmux.TGRouter) {
	router.RegisterStateHandler(State_None, h.HandleState, h.Cleanup)
}

func (h *WhoamiHandler) HandleState(ctx *tgmux.TGContext) (tgbotapi.MessageConfig, error) {
	user, _ := h.userStore.Get(fmt.Sprint(ctx.Message.Chat.ID))
	return tgbotapi.NewMessage(ctx.Message.Chat.ID, fmt.Sprintf("Your name is %v, you're a registered user", user.Username)), nil
}

func (h *WhoamiHandler) Cleanup(tgCtx *tgmux.TGContext) error {
	return nil
}
