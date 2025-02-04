package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgmux "github.com/te5se/tg-mux"
)

type NoneHandler struct {
}

func NewNoneHandler() *NoneHandler {
	return &NoneHandler{}
}

func (h *NoneHandler) Register(router *tgmux.TGRouter) {
	router.RegisterStateHandler(State_None, h.HandleState, h.Cleanup)
}

func (h *NoneHandler) HandleState(ctx *tgmux.TGContext) (tgbotapi.MessageConfig, error) {
	return tgbotapi.NewMessage(ctx.Message.Chat.ID, "Use /help command to see what you can do here"), nil
}

func (h *NoneHandler) Cleanup(tgCtx *tgmux.TGContext) error {
	return nil
}
