package config

import (
	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfighcl"
	"log"
	"sync"
	"time"
)

// Хранить в файле мы будем в формате hcl.
// Также указываем ключ для переменных окружения
type Config struct {
	TelegramBotToken     string        `hcl:"telegram_bot_token" env:"TELEGRAM_BOT_TOKEN" required:"true"`
	TelegramChannelID    int64         `hcl:"telegram_channel_id" env:"TELEGRAM_CHANNEL_ID" required:"true"`
	DatabaseDSN          string        `hcl:"database_dsn" env:"DATABASE_DSN" default:"postgres://postgres:postgres@localhost:5432/news_feed_bot?sslmode=disable"`
	FetchInterval        time.Duration `hcl:"fetch_interval" env:"FETCH_INTERVAL" default:"1m"`
	NotificationInterval time.Duration `hcl:"notification_interval" env:"NOTIFICATION_INTERVAL" default:"1m"`
	FilterKeywords       []string      `hcl:"filter_keywords" env:"FILTER_KEYWORDS"`
	OpenAIKey            string        `hcl:"openai_key" env:"OPENAI_KEY"`
	OpenAIPromt          string        `hcl:"openai_promt" env:"OPENAI_PROMT"`
}

// cfg - инстанс конфига, в который мы будем читать данные
// И once, которая нам гарантирует что функция вызванная с помощью этого примитива будем выполнена не более чем один раз.
// Это полезно поскольку мы будем тригерить чтение config, а точнее его получения из разных мест и в произвольном порядке, поэтому здесь нам нужно обеспечить эту гарантию
var (
	cfg  Config
	once sync.Once
)

// Метод get, который возвращает конфиг
func Get() Config {
	once.Do(func() {
		// Создаем лоадер
		loader := aconfig.LoaderFor(&cfg, aconfig.Config{
			// Префикс для переменных окружения, чтобы они случайно не пересеклись с какими нибудь системными или переменным окружения других программ.
			EnvPrefix: "NFB",
			// Задаем пути, где могут лежать конфиги и гле их можно искать
			Files: []string{"./config.hcl", "./config.local.hcl"},
			// Декодер для hcl
			FileDecoders: map[string]aconfig.FileDecoder{
				// Создаем инстанс для декодера этого формата
				".hcl": aconfighcl.New(),
			},
		})

		// Загрузка конфига
		if err := loader.Load(); err != nil {
			log.Printf("[ERROR] failed to load config: %v", err)
		}
	})

	return cfg
}
