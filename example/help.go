package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgmux "github.com/te5se/tg-mux"
)

const (
	HelpMessage = "Hi! Since that is the test bot, there is just one command available: /whoami."
)

type HelpHandler struct {
}

func NewHelpHandler() *HelpHandler {
	return &HelpHandler{}
}

func (h *HelpHandler) Register(router *tgmux.TGRouter) {
	router.RegisterCommand(State_Help, h.HandleState)
}

func (h *HelpHandler) HandleState(ctx *tgmux.TGContext) (tgbotapi.MessageConfig, error) {
	return tgbotapi.NewMessage(ctx.Message.Chat.ID, HelpMessage), nil
}
