package source

import (
	"context"
	"github.com/SlyMarbo/rss"
	"github.com/kovalyov-valentin/news-feed-bot/internal/model"
)

// RSS клиент.
type RSSSource struct {
	// URL откуда мы забираем данные
	URL string
	// Его id
	SourceID   int64
	SourceName string
}

// Конструктор, который будет из модели источника создавать источник уже как клиент для RSS лент
func NewRSSSourceFromModel(m model.Source) RSSSource {
	return RSSSource{
		URL:        m.FeedURL,
		SourceID:   m.ID,
		SourceName: m.Name,
	}
}

// Публичный метод, который обрабатывает данные из из лент, возвращая слайс статей
func (s RSSSource) Fetch(ctx context.Context) ([]model.Item, error) {
	// Вызываю метод, который мной уже написан
	feed, err := s.loadFeed(ctx, s.URL)
	// По хорошему ее нужно здесь во что-то заврапить
	if err != nil {
		return nil, err
	}

	//// Передаем items, и по одному мапим модельки
	//return lo.Map(feed.Items, func(item *rss.Item, _ int) model.Item {
	//	return model.Item{
	//		Title:      item.Title,
	//		Categories: item.Categories,
	//		Link:       item.Link,
	//		Date:       item.Date,
	//		Summary:    item.Summary,
	//		SourceName: s.SourceName,
	//	}
	//}), nil

	// Пример без использования либы lo
	var items []model.Item
	for _, item := range feed.Items {
		items = append(items, model.Item{
			Title:      item.Title,
			Categories: item.Categories,
			Link:       item.Link,
			Date:       item.Date,
			Summary:    item.Summary,
			SourceName: s.SourceName,
		})
	}
	return items, nil
}

// Метод, который загружает данные из источника
func (s RSSSource) loadFeed(ctx context.Context, url string) (*rss.Feed, error) {
	// Каналы для получения ленты и получения ошибок
	var (
		feedCh = make(chan *rss.Feed)
		errCh  = make(chan error)
	)

	// Запускает фетчинг ленты
	go func() {
		feed, err := rss.Fetch(url)
		if err != nil {
			errCh <- err
			return
		}

		feedCh <- feed
	}()

	select {
	// Ошибка контекста
	case <-ctx.Done():
		return nil, ctx.Err()
		// Ошибка из  канала ошибок
	case err := <-errCh:
		return nil, err
	// Если все ок, читаетм ленты из канала лент и возвращаем ленту
	case feed := <-feedCh:
		return feed, nil
	}
}

func (s RSSSource) ID() int64 {
	return s.SourceID
}

func (s RSSSource) Name() string {
	return s.SourceName
}
