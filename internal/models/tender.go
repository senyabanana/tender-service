package models

import "time"

type (
	TenderServiceType string // Тип услуги для тендера
	TenderStatus      string // Статус тендера
)

const (
	Construction TenderServiceType = "Construction"
	Delivery     TenderServiceType = "Delivery"
	Manufacture  TenderServiceType = "Manufacture"

	CreatedTender   TenderStatus = "Created"   // Тендер создан
	PublishedTender TenderStatus = "Published" // Тендер опубликован
	ClosedTender    TenderStatus = "Closed"    // Тендер закрыт
)

// Tender представляет модель тендера.
type Tender struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Status          TenderStatus      `json:"status"`
	ServiceType     TenderServiceType `json:"serviceType"`
	OrganizationID  string            `json:"organizationId"`
	Version         int32             `json:"version"`
	CreatedAt       time.Time         `json:"createdAt"`
	CreatorUsername string            `json:"-"`
}

// TenderRequest представляет структуру запроса для создания или обновления тендера.
type TenderRequest struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	ServiceType     TenderServiceType `json:"serviceType"`
	OrganizationID  string            `json:"organizationId"`
	CreatorUsername string            `json:"creatorUsername"`
}
