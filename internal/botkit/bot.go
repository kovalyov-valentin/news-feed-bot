package botkit

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"runtime/debug"
	"time"
)

type Bot struct {
	// Инстанст апи телеграма
	api *tgbotapi.BotAPI
	// Мапа в которой будем хранить view
	cmdViews map[string]ViewFunc
}

// addsource
// listsources
// deletesource

// Update здесь это любой эвент, который приходит от телеграма при взаимодействии пользователя с ботом
// инстанс botapi - это клиент через который мы получаем доступ к телеграмму
// Это функция которая будет реагировать на определенную команду
type ViewFunc func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error

func New(api *tgbotapi.BotAPI) *Bot {
	return &Bot{
		api: api,
	}
}

// Метод для регистрации View для команды
func (b *Bot) RegisterCmdView(cmd string, view ViewFunc) {
	// Проверка на то, что мапа со view инициализирована
	if b.cmdViews == nil {
		// Если нет, то ее нужно инициализировать
		b.cmdViews = make(map[string]ViewFunc)
	}

	// Добавляем в мапу эту view
	b.cmdViews[cmd] = view
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case update := <-updates:
			updateCtx, updateCancel := context.WithTimeout(ctx, 5*time.Second)
			b.handleUpdate(updateCtx, update)
			updateCancel()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Метод, который обрабатывает update и роутит команды на сооветствующие view
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// В процессе работы бота в каких то view может произойти паника, поэтому мы ее должны перехватить
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ERROR] panic recovered: %v\n%s", p, string(debug.Stack()))
		}
	}()

	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	// в эту переменную кладем view,которую мы определим как view необходимую для обработки запроса
	var view ViewFunc

	// Если сообщение не содержит никакой команды
	if !update.Message.IsCommand() {
		return
	}

	// Если это все таки команда, то нам нужно вытащить ее из сообщения, так сообщение может содержать не только команду но еще что-то
	cmd := update.Message.Command()

	// Пробуем достать view из мапы
	cmdView, ok := b.cmdViews[cmd]
	if !ok {
		return
	}

	view = cmdView

	// Вызываем view и обрабатываем ошибку от нее если та вернула ошибку
	if err := view(ctx, b.api, update); err != nil {
		log.Printf("[ERROR] failed to handle update: %v", err)

		if _, err := b.api.Send(
			tgbotapi.NewMessage(update.Message.Chat.ID, "internal error"),
		); err != nil {
			log.Printf("[ERROR] failed to send message: %v", err)
		}
	}
}
