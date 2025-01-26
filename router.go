package router

import (
	"context"
	"fmt"
	"net/http"
	"runtime"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

// https://github.com/golang/mock
type BotAPI interface {
	CopyMessage(config tgbotapi.CopyMessageConfig) (tgbotapi.MessageID, error)
	GetChat(config tgbotapi.ChatInfoConfig) (tgbotapi.Chat, error)
	GetChatAdministrators(config tgbotapi.ChatAdministratorsConfig) ([]tgbotapi.ChatMember, error)
	GetChatMember(config tgbotapi.GetChatMemberConfig) (tgbotapi.ChatMember, error)
	GetChatMembersCount(config tgbotapi.ChatMemberCountConfig) (int, error)
	GetFile(config tgbotapi.FileConfig) (tgbotapi.File, error)
	GetFileDirectURL(fileID string) (string, error)
	GetGameHighScores(config tgbotapi.GetGameHighScoresConfig) ([]tgbotapi.GameHighScore, error)
	GetInviteLink(config tgbotapi.ChatInviteLinkConfig) (string, error)
	GetMe() (tgbotapi.User, error)
	GetMyCommands() ([]tgbotapi.BotCommand, error)
	GetMyCommandsWithConfig(config tgbotapi.GetMyCommandsConfig) ([]tgbotapi.BotCommand, error)
	GetStickerSet(config tgbotapi.GetStickerSetConfig) (tgbotapi.StickerSet, error)
	GetUpdates(config tgbotapi.UpdateConfig) ([]tgbotapi.Update, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	GetUserProfilePhotos(config tgbotapi.UserProfilePhotosConfig) (tgbotapi.UserProfilePhotos, error)
	GetWebhookInfo() (tgbotapi.WebhookInfo, error)
	HandleUpdate(r *http.Request) (*tgbotapi.Update, error)
	IsMessageToMe(message tgbotapi.Message) bool
	ListenForWebhook(pattern string) tgbotapi.UpdatesChannel
	ListenForWebhookRespReqFormat(w http.ResponseWriter, r *http.Request) tgbotapi.UpdatesChannel
	MakeRequest(endpoint string, params tgbotapi.Params) (*tgbotapi.APIResponse, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	SendMediaGroup(config tgbotapi.MediaGroupConfig) ([]tgbotapi.Message, error)
	SetAPIEndpoint(apiEndpoint string)
	StopPoll(config tgbotapi.StopPollConfig) (tgbotapi.Poll, error)
	StopReceivingUpdates()
	UploadFiles(endpoint string, params tgbotapi.Params, files []tgbotapi.RequestFile) (*tgbotapi.APIResponse, error)
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
	bot             *tgbotapi.BotAPI
	commandHandlers map[string]func(ctx *TGContext) (tgbotapi.MessageConfig, error)
	stateHandlers   map[string]stateHandler
	userStateGetter func(context *TGContext) (string, error)
	queueManager    *queueManager
	ctx             context.Context
}

// userStateGetter should return a string function if user and state can be identified
// empty state string and empty error if user isn't registered yet
// and error if there was an "internal server error"
func NewTGRouter(bot *tgbotapi.BotAPI, userStateGetter func(context *TGContext) (string, error)) (*TGRouter, error) {
	var router = TGRouter{
		commandHandlers: make(map[string]func(ctx *TGContext) (tgbotapi.MessageConfig, error)),
		stateHandlers:   make(map[string]stateHandler),
		userStateGetter: userStateGetter,
		bot:             bot,
	}

	return &router, nil
}
func (router *TGRouter) Run(ctx context.Context) {
	router.ctx = ctx
	router.queueManager = newQueueManager(router.processUpdate, ctx)

	router.processUpdates()
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
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "command isn't registered")
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
	if state == "" && update.Message.Command() != "start" && update.Message.Command() != "help" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Для того, чтобы начать пользоваться ботом, выберите /start")
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

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "TODO: error text")
	_, err = router.bot.Send(msg)

	log.Err(err).Msg("while handling error")
}

func buildTGContext(msg *tgbotapi.Message) *TGContext {
	return &TGContext{
		Message: msg,
	}
}
