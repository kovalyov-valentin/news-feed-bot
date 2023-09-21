package bot

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kovalyov-valentin/news-feed-bot/internal/botkit"
	"github.com/kovalyov-valentin/news-feed-bot/internal/model"
)

type SourceStorage interface {
	Add(ctx context.Context, source model.Source) (int64, error)
}

// Метод для добавления источника в БД
func ViewCmdAddSource(storage SourceStorage) botkit.ViewFunc {
	type addSourceArgs struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		args, err := botkit.ParseJSON[addSourceArgs](update.Message.CommandArguments())
		if err != nil {
			// Нужно сказать пользаку что он взял некорректный инпут
			return err
		}

		// Воссоздаем метаинформацию об источнике из аргументов
		source := model.Source{
			Name:    args.Name,
			FeedURL: args.URL,
		}

		sourceID, err := storage.Add(ctx, source)
		if err != nil {
			return err
		}

		var (
			msgText = fmt.Sprintf(
				"Источник добавлен с ID: `%d`\\. Используйте этот ID для управления источником\\.",
				sourceID,
			)
			reply = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		)

		reply.ParseMode = "MarkdownV2"

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
