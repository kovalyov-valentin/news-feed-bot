package fetcher

import (
	"context"
	"github.com/kovalyov-valentin/news-feed-bot/internal/model"
	"github.com/kovalyov-valentin/news-feed-bot/internal/source"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tomakado/containers/set"
)

type ArticleStorage interface {
	Store(ctx context.Context, article model.Article) error
}

type SourceProvider interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

// Интерфейс источника
type Source interface {
	ID() int64
	Name() string
	// Этот метод уже реализован у RSS источника
	Fetch(ctx context.Context) ([]model.Item, error)
}

// Структура сборщика
type Fetcher struct {
	// Хранилище статей
	articles ArticleStorage
	// Хранилище источников
	sources SourceProvider

	// Как часто нам надо обновлять источники и доставать статьи
	fetchInterval time.Duration
	// Фильтрация статей по ключевым словами
	filterKeyWords []string
}

// Что то типо конструктора.
// Указываем все те параметры, который указаны как поля структуры.
// Делаем это для того, чтобы их неьзя было менять извне.
// Скрываем их, делаем не экспортируемыми и передаем в конструктор
func NewFetcher(articleStorage ArticleStorage, sourceProvider SourceProvider, fetchInterval time.Duration, filterKeyWords []string) *Fetcher {
	return &Fetcher{
		articles:       articleStorage,
		sources:        sourceProvider,
		fetchInterval:  fetchInterval,
		filterKeyWords: filterKeyWords,
	}
}

// Метод для запуски Fetcher.
// Fetcher работает в отельной горутине, как самостоятельный воркер.
// И периодически, по fetchInterval будет ходить и забирать статьи
func (f *Fetcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	if err := f.Fetch(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	// Получаем список источников через SourceProvider
	sources, err := f.sources.Sources(ctx)
	if err != nil {
		return err
	}

	// Добавляем wg, так как источников много и при опрашивании источников может что-то пойти не так.
	// Например запрос будет долго обрабатываться, долго доходить ответ или вообще что-то сломается.
	// Мы не хотим чтобы это повлияло на обработку других источников, поэтому ходить по источникам мы будем параллельно.
	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)

		rssSource := source.NewRSSSourceFromModel(src)

		// Передаем интерфейс источника. Например RSS клиент
		go func(source Source) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("[ERROR] Fetching items from source %s: %v", source.Name(), err)
				return
			}

			// Обработка items, в первую очередь сохранить их в базу
			if err := f.processItems(ctx, source, items); err != nil {
				log.Printf("[ERROR] Processing items from source %s: %v", source.Name(), err)
				return
			}
		}(rssSource)
	}

	wg.Wait()

	return nil
}

// Метод для процессинга
func (f *Fetcher) processItems(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		item.Date = item.Date.UTC()

		// Проверка item, может его нужно скипнуть
		if f.itemShouldBeSkipped(item) {
			continue
		}

		// Если все ок, сохраняем статью в ArcticleStorage
		if err := f.articles.Store(ctx, model.Article{
			SourceID:    source.ID(),
			Title:       item.Title,
			Link:        item.Link,
			Summary:     item.Summary,
			PublishedAt: item.Date,
		}); err != nil {
			return err
		}
	}

	return nil
}

// В этом методе проходимся по списку категорий, к которым относится эта статья и по title.
// Хотим выяснить есть ли ключевые слова на основе которых мы пропускаем эту статью
func (f *Fetcher) itemShouldBeSkipped(item model.Item) bool {
	// Используем сет, а не слайс,
	// чтобы быстро проверять присутствует ли ключевое слово в наборе категорий.
	// С помощью сета это делать чуть быстрее.
	// (Можно использовать стандартный сет из стандартной мапы)
	categoriesSet := set.New(item.Categories...)

	// Проходимся по каждому ключевому слову.
	for _, keyword := range f.filterKeyWords {
		titleContainsKeyword := strings.Contains(strings.ToLower(item.Title), keyword)

		// Если ключевое слово содержится в категориях или заголовках статей,
		// то мы будем пропускать эту статью
		if categoriesSet.Contains(keyword) || titleContainsKeyword {
			return true
		}
	}

	return false
}
