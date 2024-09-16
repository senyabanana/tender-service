package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/senyabanana/tender-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SendErrorResponse отправляет ошибку в формате JSON
func SendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := models.ErrorResponse{
		StatusCode: statusCode,
		Message:    message,
	}
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Println(err)
	}
}

// ParseLimitOffset обрабатывает limit и offset
func ParseLimitOffset(limitStr, offsetStr string) (int, int, error) {
	var limit, offset int
	var err error

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 50 {
			return 0, 0, fmt.Errorf("invalid limit parameter, must be a positive integer [0:50]")
		}
	} else {
		limit = 5
	}

	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("invalid offset parameter, must be a non-negative integer")
		}
	} else {
		offset = 0
	}

	return limit, offset, nil
}

// CheckOrganizationExists проверяет, существует ли организация
func CheckOrganizationExists(ctx context.Context, dbPool *pgxpool.Pool, organizationId string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organization WHERE id = $1)`
	err := dbPool.QueryRow(ctx, query, organizationId).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CheckUserInAnyOrganization проверяет, состоит ли пользователь в какой-либо организации.
func CheckUserInAnyOrganization(ctx context.Context, dbPool *pgxpool.Pool, userId string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organization_responsible WHERE user_id = $1)`
	err := dbPool.QueryRow(ctx, query, userId).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CheckUserResponsibleForOrganization проверяет, является ли пользователь ответственным за создание тендеров для организации
func CheckUserResponsibleForOrganization(ctx context.Context, dbPool *pgxpool.Pool, user, organizationId string) (bool, error) {
	var isResponsible bool
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM organization_responsible orr
			JOIN employee e ON orr.user_id = e.id
			WHERE e.username = $1 AND orr.organization_id = $2
		)`
	err := dbPool.QueryRow(ctx, query, user, organizationId).Scan(&isResponsible)
	if err != nil {
		return false, err
	}
	return isResponsible, nil
}

// CheckUserExists проверяет, существует ли пользователь с указанным username
func CheckUserExists(ctx context.Context, dbPool *pgxpool.Pool, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM employee WHERE username = $1)`
	err := dbPool.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CheckUserExistsById проверяет, существует ли пользователь по полю id
func CheckUserExistsById(ctx context.Context, dbPool *pgxpool.Pool, userId string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM employee WHERE id = $1)`
	err := dbPool.QueryRow(ctx, query, userId).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ContainsTender - функция для проверки перехода у тендеров
func ContainsTender(validTransitions []models.TenderStatus, newStatus models.TenderStatus) bool {
	for _, validStatus := range validTransitions {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}

// CheckTenderExists проверяет, существует ли тендер
func CheckTenderExists(ctx context.Context, dbPool *pgxpool.Pool, tenderId string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM tender WHERE id = $1)`
	err := dbPool.QueryRow(ctx, query, tenderId).Scan(&exists)
	return exists, err
}

// CheckUserAuthorized проверяет, что пользователь имеет право просматривать предложения по этому тендеру
func CheckUserAuthorized(ctx context.Context, dbPool *pgxpool.Pool, username, tenderId string) (bool, error) {
	var isAuthorized bool
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM tender
			WHERE id = $1 AND creator_username = $2
			OR EXISTS(
				SELECT 1 FROM organization_responsible
				WHERE organization_id = tender.organization_id
				AND user_id = (SELECT id FROM employee WHERE username = $2)
			)
		)`
	err := dbPool.QueryRow(ctx, query, tenderId, username).Scan(&isAuthorized)
	return isAuthorized, err
}

// CheckUserAuthorizedForBid проверяет, что пользователь имеет право просматривать заявку (bid) по этому bidId.
func CheckUserAuthorizedForBid(ctx context.Context, dbPool *pgxpool.Pool, username, bidId string) (bool, error) {
	var isAuthorized bool
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM bid
			WHERE id = $1
			AND author_id = (SELECT id FROM employee WHERE username = $2)
		)`
	err := dbPool.QueryRow(ctx, query, bidId, username).Scan(&isAuthorized)
	return isAuthorized, err
}

// ContainsBid - функция для проверки перехода у предложений
func ContainsBid(validStatuses []models.BidStatus, newStatus models.BidStatus) bool {
	for _, validStatus := range validStatuses {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}

// CheckBidExists проверяет существование предложения по его ID
func CheckBidExists(ctx context.Context, dbPool *pgxpool.Pool, bidId string) (bool, error) {
	var exists bool
	bidQuery := `SELECT EXISTS(SELECT 1 FROM bid WHERE id = $1)`
	err := dbPool.QueryRow(ctx, bidQuery, bidId).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetTenderById получает тендер по ID.
func GetTenderById(ctx context.Context, dbPool *pgxpool.Pool, tenderId string) (*models.Tender, error) {
	var tender models.Tender
	query := `SELECT id, name, description, service_type, status, organization_id, version, created_at, creator_username
	          FROM tender WHERE id = $1`
	err := dbPool.QueryRow(ctx, query, tenderId).Scan(
		&tender.ID,
		&tender.Name,
		&tender.Description,
		&tender.ServiceType,
		&tender.Status,
		&tender.OrganizationID,
		&tender.Version,
		&tender.CreatedAt,
		&tender.CreatorUsername,
	)
	if err != nil {
		return nil, err
	}
	return &tender, nil
}

// GetBidById получает заявку (bid) по ID.
func GetBidById(ctx context.Context, dbPool *pgxpool.Pool, bidId string) (*models.Bid, error) {
	var bid models.Bid
	query := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
	          FROM bid WHERE id = $1`
	err := dbPool.QueryRow(ctx, query, bidId).Scan(
		&bid.ID,
		&bid.Name,
		&bid.Description,
		&bid.Status,
		&bid.TenderId,
		&bid.AuthorType,
		&bid.AuthorId,
		&bid.Version,
		&bid.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &bid, nil
}
