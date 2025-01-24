package router

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type userMessageQueue struct {
	updateChan    chan tgbotapi.Update
	processUpdate func(tgbotapi.Update)
	stopped       bool
	ctx           context.Context
}

func newUserMessageQueue(processFunc func(tgbotapi.Update), ctx context.Context) *userMessageQueue {
	return &userMessageQueue{
		updateChan:    make(chan tgbotapi.Update, 5),
		processUpdate: processFunc,
		ctx:           ctx,
	}
}
func (q *userMessageQueue) Start() {
	go func() {
		for {
			select {
			case update := <-q.updateChan:
				q.processUpdate(update)
			case <-q.ctx.Done():
				return
			case <-time.After(time.Second * 10):
				q.stopped = true
				return
			}
		}
	}()
}

func (q *userMessageQueue) Add(update tgbotapi.Update) {
	q.updateChan <- update
}
