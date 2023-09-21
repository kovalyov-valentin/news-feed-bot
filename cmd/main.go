package main

import (
	"context"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"github.com/kovalyov-valentin/news-feed-bot/internal/bot"
	"github.com/kovalyov-valentin/news-feed-bot/internal/bot/middleware"
	"github.com/kovalyov-valentin/news-feed-bot/internal/botkit"
	"github.com/kovalyov-valentin/news-feed-bot/internal/config"
	"github.com/kovalyov-valentin/news-feed-bot/internal/fetcher"
	"github.com/kovalyov-valentin/news-feed-bot/internal/notifier"
	"github.com/kovalyov-valentin/news-feed-bot/internal/storage"
	"github.com/kovalyov-valentin/news-feed-bot/internal/summary"
	_ "github.com/lib/pq"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Создаем бота, используя токен из конфига
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Printf("failed to create bot: %v", err)
		return
	}

	// Инициализируем подключение к БД
	db, err := sqlx.Connect("postgres", config.Get().DatabaseDSN)
	if err != nil {
		log.Printf("failed to connect to database: %v", err)
		return
	}
	defer db.Close()

	// Инициализируем наши зависимости
	var (
		articleStorage = storage.NewArticleStorage(db)
		sourceStorage  = storage.NewSourcePostgresStorage(db)
		fetcher        = fetcher.NewFetcher(
			articleStorage,
			sourceStorage,
			config.Get().FetchInterval,
			config.Get().FilterKeywords,
		)
		notifier = notifier.New(
			articleStorage,
			summary.NewOpenAISummarizer(config.Get().OpenAIKey, config.Get().OpenAIPromt),
			botAPI,
			// Интервал отправки сообщений
			config.Get().NotificationInterval,
			// Интервал которым мы будем заглядывать в прошлое (lookapthewindow)
			2*config.Get().FetchInterval,
			config.Get().TelegramChannelID,
		)
	)

	//Graceful Shatdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Инициализируем нашего бота
	// Обернуть middleware все view где нужно дать доступ только админу
	newsBot := botkit.New(botAPI)
	newsBot.RegisterCmdView("start", bot.ViewCmdStart())
	newsBot.RegisterCmdView(
		"addsource",
		middleware.AdminOnly(
			config.Get().TelegramChannelID,
			bot.ViewCmdAddSource(sourceStorage),
		),
	)
	newsBot.RegisterCmdView("listsources", bot.ViewCmdListSources(sourceStorage))

	// Воркер fetcher
	go func(ctx context.Context) {
		if err := fetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to start fetcher: %v", err)
				return
			}

			log.Println("fetcher stopped")
		}
	}(ctx)

	// Воркер notifier
	go func(ctx context.Context) {
		if err := notifier.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to start notifier: %v", err)
				return
			}

			log.Println("notifier stopped")
		}
	}(ctx)

	// Запуск бота
	if err := newsBot.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Printf("[ERROR] failed to start bot: %v", err)
			return
		}

		log.Println("bot stopped")
	}

}
