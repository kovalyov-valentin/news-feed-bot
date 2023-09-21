package model

import "time"

// Статья как элемент ленты
type Item struct {
	// Название статьи
	Title string
	// Категории статей
	Categories []string
	// Ссылка
	Link string
	// Дата публикации в источнике
	Date time.Time
	// Краткая выжимка
	Summary string
	// Имя источника
	SourceName string
}

// Модель источника
type Source struct {
	ID int64
	// Имя
	Name string
	// Урл откуда забираем данные
	FeedURL string
	//Priority  int
	// Время создания
	CreatedAt time.Time
}

// Модель статьи которая используется у нас внутри а не в RSS
type Article struct {
	ID       int64
	SourceID int64
	Title    string
	Link     string
	Summary  string
	// Время публикации в источнике
	PublishedAt time.Time
	// Время публикации в телеграмм канале
	PostedAt time.Time
	// Время создания
	CreatedAt time.Time
}
