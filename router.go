package tgmux

import (
	"context"
	"fmt"
	"runtime"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

type BotAPI interface {
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	StopReceivingUpdates()
}

type TGContext struct {
	Message *tgbotapi.Message
}

type stateHandler struct {
	Handler func(ctx *TGContext) (tgbotapi.MessageConfig, error)
	Cleanup func(tgCtx *TGContext) error
}

type TGRouter struct {
	tgKey           string
	bot             BotAPI
	commandHandlers map[string]func(ctx *TGContext) (tgbotapi.MessageConfig, error)
	stateHandlers   map[string]stateHandler
	userStateGetter func(context *TGContext) (string, error)
	queueManager    *queueManager
	ctx             context.Context
	localization    Localization
}

// userStateGetter should return a string function if user and state can be identified
// empty state string and empty error if user isn't registered yet
// and error if there was an "internal server error"
func NewTGRouter(bot BotAPI, userStateGetter func(context *TGContext) (string, error)) (*TGRouter, error) {
	var router = TGRouter{
		commandHandlers: make(map[string]func(ctx *TGContext) (tgbotapi.MessageConfig, error)),
		stateHandlers:   make(map[string]stateHandler),
		userStateGetter: userStateGetter,
		bot:             bot,
		localization:    defaultLocalization,
	}

	return &router, nil
}
func (router *TGRouter) Run(ctx context.Context) {
	router.ctx = ctx
	router.queueManager = newQueueManager(router.processUpdate, ctx)

	router.processUpdates()
}

func (router *TGRouter) ConfigureMessages(localization Localization) {
	router.localization = localization
}

func (router *TGRouter) RegisterCommand(name string, handleFunc func(ctx *TGContext) (tgbotapi.MessageConfig, error)) {
	router.commandHandlers[name] = handleFunc
}

// cleanupFunc allows to roll back changes if your flow was interrupted by another command
func (router *TGRouter) RegisterStateHandler(state string, handleFunc func(ctx *TGContext) (tgbotapi.MessageConfig, error), cleanupFunc func(tgCtx *TGContext) error) {
	router.stateHandlers[state] = stateHandler{
		Handler: handleFunc,
		Cleanup: cleanupFunc,
	}
}

func (router *TGRouter) processUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := router.bot.GetUpdatesChan(u)

	for {
		select {
		case <-router.ctx.Done():
			router.bot.StopReceivingUpdates()
			return
		case update := <-updates:
			queue := router.queueManager.GetOrCreateQueue(fmt.Sprint(update.Message.Chat.ID))
			queue.Add(update)
		}
	}
}

func (router *TGRouter) processUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			stackSize := 1024 * 8

			stack := make([]byte, stackSize)
			stack = stack[:runtime.Stack(stack, false)]

			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}

			log.Debug().Err(err).Str("stack", string(stack)).Msg("error message")
			router.handleError(err, "panicked", update)
		}
	}()

	if update.Message == nil {
		return
	}

	log.Debug().Msgf("[%s] %s %s", update.Message.From.UserName, update.Message.Text, update.Message.Command())

	state, err := router.userStateGetter(buildTGContext(update.Message))
	if err != nil {
		router.handleError(err, "while getting user from repository", update)
		return
	}
	shouldGoOn := router.handleNonExistentUsers(update)
	if !shouldGoOn {
		return
	}

	if update.Message.Command() != "" {
		// Unfinished state cleanup before command execution
		if state != "" {
			cleanupHandler, ok := router.stateHandlers[state]
			if !ok {
				router.handleError(fmt.Errorf("handler %v isn't registered", state), "", update)
				return
			}

			err := cleanupHandler.Cleanup(buildTGContext(update.Message))
			if err != nil {
				router.handleError(err, "while handling cleanup", update)
				return
			}
		}

		// Command handling
		commandHandler, ok := router.commandHandlers[update.Message.Command()]
		if !ok {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, router.localization.CommandNotFound)
			router.bot.Send(msg)
			return
		}

		msg, err := commandHandler(buildTGContext(update.Message))
		if err != nil {
			router.handleError(err, "while handling command", update)
			return
		}

		router.bot.Send(msg)
		return
	}

	stateHandler, ok := router.stateHandlers[state]
	if !ok {
		router.handleError(fmt.Errorf("handler %v isn't registered", state), "", update)
		return
	}

	msg, err := stateHandler.Handler(buildTGContext(update.Message))
	if err != nil {
		router.handleError(err, fmt.Sprintf("while using state handler %v", state), update)
		return
	}

	router.bot.Send(msg)
	return
}

func (router *TGRouter) handleNonExistentUsers(update tgbotapi.Update) (shouldContinue bool) {
	state, err := router.userStateGetter(buildTGContext(update.Message))
	if err != nil {
		router.handleError(err, "while getting state from repository", update)
		return false
	}
	if state == "" && update.Message.Command() != "start" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, router.localization.UseStartToRegister)
		_, err = router.bot.Send(msg)
		if err != nil {
			log.Err(err).Msg("while sending msg to TG")
			return false
		}

		return false
	}

	return true
}

func (router *TGRouter) sendWithLogErr(message *tgbotapi.MessageConfig) {
	_, err := router.bot.Send(message)
	if err != nil {
		log.Err(err).Msg("while sending msg to TG")
	}
}

func (router *TGRouter) handleError(err error, errMessage string, update tgbotapi.Update) {
	log.Err(err).Msg(errMessage)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, router.localization.OnError)
	_, err = router.bot.Send(msg)

	log.Err(err).Msg("while handling error")
}

func buildTGContext(msg *tgbotapi.Message) *TGContext {
	return &TGContext{
		Message: msg,
	}
}
