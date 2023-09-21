package middleware

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kovalyov-valentin/news-feed-bot/internal/botkit"
)

func AdminOnly(channelID int64, next botkit.ViewFunc) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		admins, err := bot.GetChatAdministrators(
			tgbotapi.ChatAdministratorsConfig{
				ChatConfig: tgbotapi.ChatConfig{
					ChatID: channelID,
				},
			},
		)
		if err != nil {
			return err
		}

		// Проверка на то, что тот кто отправил команду находится в списке администраторов
		for _, admin := range admins {
			if admin.User.ID == update.Message.From.ID {
				return next(ctx, bot, update)
			}
		}

		if _, err := bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет прав для выполнения этой команды")); err != nil {
			return err
		}
		return nil
	}
}
