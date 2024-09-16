package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/senyabanana/tender-service/internal/models"
	"github.com/senyabanana/tender-service/internal/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

// TenderRepository - интерфейс для работы с тендерами.
type TenderRepository interface {
	GetTenders(ctx context.Context, limit, offset int, serviceTypes []string) ([]models.Tender, error)
	CreateTender(ctx context.Context, tenderReq models.TenderRequest) (*models.Tender, error)
	GetUserTender(ctx context.Context, limit, offset int, username string) ([]models.Tender, error)
	GetTenderStatus(ctx context.Context, tenderId, username string) (models.TenderStatus, error)
	UpdateTenderStatus(ctx context.Context, tenderId, status string) (*models.Tender, error)
	EditTender(ctx context.Context, tenderId string, updateFields map[string]interface{}) (*models.Tender, error)
	RollbackTender(ctx context.Context, tenderId string, version int) (*models.Tender, error)
}

// PostgresTenderRepository - реализация TenderRepository для базы данных.
type PostgresTenderRepository struct {
	DB *pgxpool.Pool
}

// NewPostgresTenderRepository создаёт новый экземпляр PostgresTenderRepository.
func NewPostgresTenderRepository(db *pgxpool.Pool) *PostgresTenderRepository {
	return &PostgresTenderRepository{DB: db}
}

// GetTenders возвращает список тендеров.
func (r *PostgresTenderRepository) GetTenders(ctx context.Context, limit, offset int, serviceTypes []string) ([]models.Tender, error) {
	query := `SELECT id, name, description, status, service_type, organization_id, version, created_at, creator_username FROM tender` // TODO: не забыть убрать
	var filters []string
	var args []interface{}
	argIndex := 1

	if len(serviceTypes) > 0 {
		filters = append(filters, fmt.Sprintf("service_type = ANY($%d)", argIndex))
		args = append(args, pq.Array(serviceTypes))
		argIndex++
	}

	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY name LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenders []models.Tender
	for rows.Next() {
		var tender models.Tender
		if err := rows.Scan(
			&tender.ID,
			&tender.Name,
			&tender.Description,
			&tender.ServiceType,
			&tender.Status,
			&tender.OrganizationID,
			&tender.Version,
			&tender.CreatedAt,
			&tender.CreatorUsername); err != nil {
			return nil, err
		}
		tenders = append(tenders, tender)
	}
	return tenders, nil
}

// CreateTender создает новый тендер.
func (r *PostgresTenderRepository) CreateTender(ctx context.Context, tenderReq models.TenderRequest) (*models.Tender, error) {
	newTender := models.Tender{
		ID:              uuid.New().String(),
		Name:            tenderReq.Name,
		Description:     tenderReq.Description,
		ServiceType:     tenderReq.ServiceType,
		Status:          models.CreatedTender,
		OrganizationID:  tenderReq.OrganizationID,
		Version:         1,
		CreatedAt:       time.Now().UTC(),
		CreatorUsername: tenderReq.CreatorUsername,
	}
	_, err := r.DB.Exec(ctx, `
       INSERT INTO tender (id, name, description, service_type, status, organization_id, version, created_at, creator_username)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
   `,
		newTender.ID,
		newTender.Name,
		newTender.Description,
		newTender.ServiceType,
		newTender.Status,
		newTender.OrganizationID,
		newTender.Version,
		newTender.CreatedAt,
		newTender.CreatorUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to insert tender: %w", err)
	}
	return &newTender, nil
}

// GetUserTender возвращает список тендеров для пользователя.
func (r *PostgresTenderRepository) GetUserTender(ctx context.Context, limit, offset int, username string) ([]models.Tender, error) {
	query := `SELECT id, name, description, service_type, status, organization_id, version, created_at, creator_username
              FROM tender WHERE creator_username = $1 ORDER BY name LIMIT $2 OFFSET $3`

	rows, err := r.DB.Query(ctx, query, username, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenders []models.Tender
	for rows.Next() {
		var t models.Tender
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&t.ServiceType,
			&t.Status,
			&t.OrganizationID,
			&t.Version,
			&t.CreatedAt,
			&t.CreatorUsername); err != nil {
			return nil, err
		}

		isResponsible, err := utils.CheckUserResponsibleForOrganization(ctx, r.DB, username, t.OrganizationID)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check responsibility")
		}
		if !isResponsible {
			return nil, models.NewErrorResponse(http.StatusForbidden, "you do not have permission to view tenders for this organization")
		}
		tenders = append(tenders, t)
	}
	return tenders, nil
}

// GetTenderStatus возвращает статус тендера.
func (r *PostgresTenderRepository) GetTenderStatus(ctx context.Context, tenderId, username string) (models.TenderStatus, error) {
	var status models.TenderStatus
	var query string
	var args []interface{}

	if username == "" {
		query = `SELECT status FROM tender WHERE id = $1`
		args = append(args, tenderId)
	} else {
		query = `SELECT status FROM tender WHERE id = $1 AND creator_username = $2`
		args = append(args, tenderId, username)
	}

	err := r.DB.QueryRow(ctx, query, args...).Scan(&status)
	if err != nil {
		return "", err
	}

	return status, nil
}

// UpdateTenderStatus меняет статус тендера.
func (r *PostgresTenderRepository) UpdateTenderStatus(ctx context.Context, tenderId, status string) (*models.Tender, error) {
	updateQuery := `UPDATE tender SET status = $1 WHERE id = $2`
	_, err := r.DB.Exec(ctx, updateQuery, status, tenderId)
	if err != nil {
		return nil, err
	}

	var updatedTender models.Tender
	selectQuery := `SELECT id, name, description, service_type, status, organization_id, version, created_at, creator_username
	                FROM tender WHERE id = $1`
	err = r.DB.QueryRow(ctx, selectQuery, tenderId).Scan(
		&updatedTender.ID,
		&updatedTender.Name,
		&updatedTender.Description,
		&updatedTender.ServiceType,
		&updatedTender.Status,
		&updatedTender.OrganizationID,
		&updatedTender.Version,
		&updatedTender.CreatedAt,
		&updatedTender.CreatorUsername,
	)
	if err != nil {
		return nil, err
	}
	return &updatedTender, nil
}

// EditTender меняет описание тендера.
func (r *PostgresTenderRepository) EditTender(ctx context.Context, tenderId string, updateFields map[string]interface{}) (*models.Tender, error) {
	var currentTender models.Tender
	selectQuery := `SELECT id, name, description, service_type, status, organization_id, version, created_at, creator_username
	                FROM tender WHERE id = $1`
	err := r.DB.QueryRow(ctx, selectQuery, tenderId).Scan(
		&currentTender.ID,
		&currentTender.Name,
		&currentTender.Description,
		&currentTender.ServiceType,
		&currentTender.Status,
		&currentTender.OrganizationID,
		&currentTender.Version,
		&currentTender.CreatedAt,
		&currentTender.CreatorUsername,
	)
	if err != nil {
		return nil, err
	}

	var maxVersion int
	versionQuery := `SELECT COALESCE(MAX(version), 0) FROM tender_history WHERE id = $1`
	err = r.DB.QueryRow(ctx, versionQuery, currentTender.ID).Scan(&maxVersion)
	if err != nil {
		return nil, err
	}

	historyInsertQuery := `INSERT INTO tender_history (id, name, description, service_type, status, organization_id, version, created_at, creator_username)
                      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = r.DB.Exec(
		ctx,
		historyInsertQuery,
		currentTender.ID,
		currentTender.Name,
		currentTender.Description,
		currentTender.ServiceType,
		currentTender.Status,
		currentTender.OrganizationID,
		maxVersion+1,
		currentTender.CreatedAt,
		currentTender.CreatorUsername)
	if err != nil {
		return nil, err
	}

	updateQuery := `UPDATE tender SET `
	var updates []string
	var args []interface{}
	argIndex := 1

	if name, ok := updateFields["name"].(string); ok && name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	if description, ok := updateFields["description"].(string); ok && description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, description)
		argIndex++
	}

	if serviceType, ok := updateFields["serviceType"].(string); ok && serviceType != "" {
		allowedServiceTypes := map[models.TenderServiceType]bool{
			models.Construction: true,
			models.Delivery:     true,
			models.Manufacture:  true,
		}
		if !allowedServiceTypes[models.TenderServiceType(serviceType)] {
			return nil, models.NewErrorResponse(http.StatusBadRequest, fmt.Sprintf("invalid service_type parameter: %s", serviceType))
		}
		updates = append(updates, fmt.Sprintf("service_type = $%d", argIndex))
		args = append(args, serviceType)
		argIndex++
	}

	if len(updates) == 0 {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "No valid fields to update")
	}

	updates = append(updates, fmt.Sprintf("version = version + 1"))

	updateQuery += strings.Join(updates, ", ") + fmt.Sprintf(" WHERE id = $%d RETURNING id, name, description, service_type, status, organization_id, version, created_at, creator_username", argIndex)
	args = append(args, tenderId)

	var updatedTender models.Tender
	err = r.DB.QueryRow(ctx, updateQuery, args...).Scan(
		&updatedTender.ID,
		&updatedTender.Name,
		&updatedTender.Description,
		&updatedTender.ServiceType,
		&updatedTender.Status,
		&updatedTender.OrganizationID,
		&updatedTender.Version,
		&updatedTender.CreatedAt,
		&updatedTender.CreatorUsername,
	)
	if err != nil {
		return nil, err
	}
	return &updatedTender, nil
}

// RollbackTender откатывает версию тендера
func (r *PostgresTenderRepository) RollbackTender(ctx context.Context, tenderId string, version int) (*models.Tender, error) {
	var organizationId string
	query := `SELECT organization_id FROM tender WHERE id = $1`
	err := r.DB.QueryRow(ctx, query, tenderId).Scan(&organizationId)
	if err != nil {
		return nil, err
	}

	var rollbackVersion models.Tender
	query = `SELECT id, name, description, service_type, status, organization_id, version, created_at, creator_username
	         FROM tender_history WHERE id = $1 AND version = $2`
	err = r.DB.QueryRow(ctx, query, tenderId, version).Scan(
		&rollbackVersion.ID,
		&rollbackVersion.Name,
		&rollbackVersion.Description,
		&rollbackVersion.ServiceType,
		&rollbackVersion.Status,
		&rollbackVersion.OrganizationID,
		&rollbackVersion.Version,
		&rollbackVersion.CreatedAt,
		&rollbackVersion.CreatorUsername,
	)
	if err != nil {
		return nil, err
	}

	updateQuery := `UPDATE tender SET name = $1, description = $2, service_type = $3, status = $4, version = version + 1 WHERE id = $5 RETURNING id, name, description, service_type, status, organization_id, version, created_at, creator_username`
	var updatedTender models.Tender
	err = r.DB.QueryRow(
		ctx,
		updateQuery,
		rollbackVersion.Name,
		rollbackVersion.Description,
		rollbackVersion.ServiceType,
		rollbackVersion.Status,
		tenderId).Scan(
		&updatedTender.ID,
		&updatedTender.Name,
		&updatedTender.Description,
		&updatedTender.ServiceType,
		&updatedTender.Status,
		&updatedTender.OrganizationID,
		&updatedTender.Version,
		&updatedTender.CreatedAt,
		&updatedTender.CreatorUsername,
	)
	if err != nil {
		return nil, err
	}

	historyInsertQuery := `INSERT INTO tender_history (id, name, description, service_type, status, organization_id, version, created_at, creator_username)
	                       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = r.DB.Exec(ctx, historyInsertQuery, updatedTender.ID, updatedTender.Name, updatedTender.Description, updatedTender.ServiceType, updatedTender.Status, updatedTender.OrganizationID, updatedTender.Version, updatedTender.CreatedAt, updatedTender.CreatorUsername)
	if err != nil {
		return nil, err
	}
	return &updatedTender, nil
}
