package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgmux "github.com/te5se/tg-mux"
)

type HelpHandler struct {
}

func NewHelpHandler() *HelpHandler {
	return &HelpHandler{}
}

func (h *HelpHandler) Register(router *tgmux.TGRouter) {
	router.RegisterStateHandler(State_None, h.HandleState, h.Cleanup)
}

func (h *HelpHandler) HandleState(ctx *tgmux.TGContext) (tgbotapi.MessageConfig, error) {
	return tgbotapi.NewMessage(ctx.Message.Chat.ID, "Hi! Since that is the test bot, there is just one command available: /whoami."), nil
}

func (h *HelpHandler) Cleanup(tgCtx *tgmux.TGContext) error {
	return nil
}
