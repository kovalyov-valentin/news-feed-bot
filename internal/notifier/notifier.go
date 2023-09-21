package notifier

import (
	"context"
	"fmt"
	"github.com/go-shiori/go-readability"
	"github.com/kovalyov-valentin/news-feed-bot/internal/botkit/markup"
	"github.com/kovalyov-valentin/news-feed-bot/internal/model"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ArticleProvider interface {
	AllNotPosted(ctx context.Context, since time.Time, limit uint64) ([]model.Article, error)
	MarkPosted(ctx context.Context, id int64) error
}

type Summarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

type Notifier struct {
	// Провайдер для статей
	articles ArticleProvider
	// Компонент, который будет генерить summary
	summarizer Summarizer
	// Инстанс клиента botAPI
	bot *tgbotapi.BotAPI
	// Интервал, с которым notifier будет проверять есть ли новые статьи
	sendInterval time.Duration
	// Время в прошлое, в которое будет заглядываться notifier, чтобы узнать есть за этот период новые статьи
	lookupTimeWindow time.Duration
	// id канала куда мы будем постить статьи
	channelID int64
}

func New(
	articleProvider ArticleProvider,
	summarizer Summarizer,
	bot *tgbotapi.BotAPI,
	sendInterval time.Duration,
	lookupTimeWindow time.Duration,
	channelID int64,
) *Notifier {
	return &Notifier{
		articles:         articleProvider,
		summarizer:       summarizer,
		bot:              bot,
		sendInterval:     sendInterval,
		lookupTimeWindow: lookupTimeWindow,
		channelID:        channelID,
	}
}

func (n *Notifier) Start(ctx context.Context) error {
	ticker := time.NewTicker(n.sendInterval)
	defer ticker.Stop()

	if err := n.SelectAndSendArticle(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			if err := n.SelectAndSendArticle(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Метод для выборки и отправки статьи
func (n *Notifier) SelectAndSendArticle(ctx context.Context) error {
	topOneArticles, err := n.articles.AllNotPosted(ctx, time.Now().Add(-n.lookupTimeWindow), 1)
	// Заврапить!
	if err != nil {
		return err
	}

	// ОБЕРНУТЬ ВСЕ ДЕЙСТВИЯ ЗДЕСБ В ТРАНЗАКЦИЮ

	// Если нет статьи, то ничего не делаем
	if len(topOneArticles) == 0 {
		return nil
	}

	article := topOneArticles[0]

	summary, err := n.extractSummary(ctx, article)
	if err != nil {
		return err
	}

	if err := n.sendArticle(article, summary); err != nil {
		return err
	}

	// После того, как все получилось, отмечаем статью, как запощенную
	return n.articles.MarkPosted(ctx, article.ID)
	//summary, err := n.summarizer.Summarize(ctx, article.Summary)
}

// Краткое содержание выдержки
// Есть у статьи есть summary, то мы будем использовать этот текст для gpt
// Если summary не заполнено, то мы идем по link, получаем html код страницы со статьей, и на основе этой страницы получить summary
func (n *Notifier) extractSummary(ctx context.Context, article model.Article) (string, error) {
	// Reader из которого мы в итоге будем читать summary
	var r io.Reader

	if article.Summary != "" {
		r = strings.NewReader(article.Summary)
	} else {
		// Настроить retry, back off
		resp, err := http.Get(article.Link)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		r = resp.Body
	}

	// Преобразуем наш reader в документ
	doc, err := readability.FromReader(r, nil)
	if err != nil {
		return "", err
	}

	// Получаем summary
	summary, err := n.summarizer.Summarize(ctx, cleanText(doc.TextContent))
	if err != nil {
		return "", err
	}

	return "\n\n" + summary, nil
}

// Метод отправки статьи
func (n *Notifier) sendArticle(article model.Article, summary string) error {
	// Шаблон сообщения. Сначала идет жирным заголовок, потом summary, потом ссылка на статью
	const msgFormat = "*%s*%s\n\n%s"

	msg := tgbotapi.NewMessage(n.channelID, fmt.Sprintf(
		// Т.к. используется markdown верстка и некоторые спец символы из markdown используются как обычные символы
		// Поэтому надо обернуть аргументы в escape
		msgFormat,
		markup.EscapeForMarkdown(article.Title),
		markup.EscapeForMarkdown(summary),
		markup.EscapeForMarkdown(article.Link),
	))
	// Даем понять телеграм, чтобы это сообщение парсилось как markdown сообщение
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	// Отправляем сообщение
	_, err := n.bot.Send(msg)
	if err != nil {
		return err
	}
	return nil

}

// Библиотека readability создаем много пустых строк в тексте очищенном от html тегов
// Эта регулярка соотвествует всем последовательностям пустых строк, где пустые строки идут от 3 подряд раз
// И все такие последовательности заменяем на 1 пустую строку
var redundantNewLines = regexp.MustCompile(`\n{3,}`)

// При помощи регулярки избавляемся от лишних пустых строк
func cleanText(text string) string {
	return redundantNewLines.ReplaceAllString(text, "\n")
}
