package models

import "time"

type (
	BidStatus     string // Статус предложения
	BidAuthorType string // Автор предложения
	BidDecision   string // Решение по предложению
)

const (
	Organization BidAuthorType = "Organization" // Предложение создала организация
	User         BidAuthorType = "User"         // Предложение создал пользователь

	CreatedBid   BidStatus = "Created"   // Предложение создано
	PublishedBid BidStatus = "Published" // Предложение опубликовано
	CanceledBid  BidStatus = "Canceled"  // Предложение отменено

	ApprovedBid BidDecision = "Approved" // Предложение одобрено
	RejectedBid BidDecision = "Rejected" // Предложение отклонено
)

// Bid представляет модель предложения.
type Bid struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      BidStatus     `json:"status"`
	TenderId    string        `json:"tenderId"`
	AuthorType  BidAuthorType `json:"authorType"`
	AuthorId    string        `json:"authorId"`
	Version     int           `json:"version"`
	CreatedAt   time.Time     `json:"createdAt"`
}

// BidRequest представляет структуру запроса для создания или обновления предложения.
type BidRequest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	TenderId    string        `json:"tenderId"`
	AuthorType  BidAuthorType `json:"authorType"`
	AuthorId    string        `json:"authorId"`
}

// BidReview представляет модель отзывов по предложению.
type BidReview struct {
	ID          string    `json:"id"`
	BidID       string    `json:"-"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}
