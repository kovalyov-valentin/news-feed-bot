package bot

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kovalyov-valentin/news-feed-bot/internal/botkit"
	"github.com/kovalyov-valentin/news-feed-bot/internal/botkit/markup"
	"github.com/kovalyov-valentin/news-feed-bot/internal/model"
	"github.com/samber/lo"
	"strings"
)

type SourceLister interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

func ViewCmdListSources(lister SourceLister) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		// Получаем список источников от листера
		sources, err := lister.Sources(ctx)
		if err != nil {
			return err
		}

		var (
			// Складываем в нее сформатированные тексты с метаинформацией об источниках
			sourceInfos = lo.Map(sources, func(source model.Source, _ int) string {
				return formatSource(source)
			})
			msgText = fmt.Sprintf(
				"Список источников \\(всего %d\\):\n\n%s",
				len(sources),
				strings.Join(sourceInfos, "\n\n"),
			)
		)

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		reply.ParseMode = "MarkdownV2"

		if _, err := bot.Send(reply); err != nil {
			return err
		}
		return nil

	}
}

// Вывод форматированной информации об источниках
func formatSource(source model.Source) string {
	return fmt.Sprintf(
		"🌐 *%s*\nID: `%d`\nURL фида: %s",
		markup.EscapeForMarkdown(source.Name),
		source.ID,
		markup.EscapeForMarkdown(source.FeedURL),
	)
}
