package tgmux

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotAPIMock struct {
	updChannel  chan tgbotapi.Update
	sendChannel chan tgbotapi.MessageConfig
}

func NewBotAPIMock() *BotAPIMock {
	return &BotAPIMock{
		updChannel:  make(chan tgbotapi.Update),
		sendChannel: make(chan tgbotapi.MessageConfig),
	}
}

// GetUpdatesChan implements BotAPI.
func (b BotAPIMock) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return b.updChannel
}

// Send implements BotAPI.
func (b BotAPIMock) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	msg, ok := c.(tgbotapi.MessageConfig)
	if !ok {
		log.Fatalf("not a message: %v", c)
	}
	b.sendChannel <- msg

	return tgbotapi.Message{}, nil
}

// StopReceivingUpdates implements BotAPI.
func (b BotAPIMock) StopReceivingUpdates() {
	close(b.updChannel)
}

func (b BotAPIMock) PushUpdate(update tgbotapi.Update) {
	b.updChannel <- update
}

func (b BotAPIMock) GetRecordedChan() chan tgbotapi.MessageConfig {
	return b.sendChannel
}

var _ BotAPI = BotAPIMock{}
