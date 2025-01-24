package router

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/patrickmn/go-cache"
)

type queueManager struct {
	cache       *cache.Cache
	processFunc func(tgbotapi.Update)
	ctx         context.Context
}

func newQueueManager(processFunc func(tgbotapi.Update), ctx context.Context) *queueManager {
	manager := queueManager{
		cache:       cache.New(-1, -1),
		processFunc: processFunc,
		ctx:         ctx,
	}

	go manager.cleanup()

	return &manager
}

func (m *queueManager) GetOrCreateQueue(chatID string) *userMessageQueue {
	queue, ok := m.cache.Get(chatID)
	if !ok {
		messageQueue := newUserMessageQueue(m.processFunc, m.ctx)
		messageQueue.Start()

		m.cache.Set(chatID, messageQueue, -1)

		return messageQueue
	}
	msgQueue := queue.(*userMessageQueue)
	if msgQueue.stopped {
		messageQueue := newUserMessageQueue(m.processFunc, m.ctx)
		messageQueue.Start()

		m.cache.Set(chatID, messageQueue, -1)

		return messageQueue
	}

	return msgQueue
}

func (m *queueManager) cleanup() {
	go func() {
		for {
			time.Sleep(time.Minute * 10)

			for key, item := range m.cache.Items() {
				if item.Object.(*userMessageQueue).stopped {
					m.cache.Delete(key)
				}
			}
		}
	}()
}
