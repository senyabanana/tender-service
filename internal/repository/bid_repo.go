package repository

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/senyabanana/tender-service/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BidRepository - интерфейс для работы с предложениями.
type BidRepository interface {
	CreateBid(ctx context.Context, bidReq models.BidRequest) (*models.Bid, error)
	GetUserBid(ctx context.Context, limit, offset int, username string) ([]models.Bid, error)
	GetTenderBid(ctx context.Context, tenderId string, limit, offset int) ([]models.Bid, error)
	GetBidStatus(ctx context.Context, bidId string) (*models.BidStatus, error)
	UpdateBidStatus(ctx context.Context, bidId, status string) (*models.Bid, error)
	EditBid(ctx context.Context, bidId string, updateFields map[string]interface{}) (*models.Bid, error)
	SubmitBidDecision(ctx context.Context, bidId, decision string) (*models.Bid, error)
	SubmitBidFeedback(ctx context.Context, review models.BidReview, bidId string) (*models.Bid, error)
	RollbackBid(ctx context.Context, bidId string, version int) (*models.Bid, error)
	GetBidReviews(ctx context.Context, tenderId, authorUsername, requesterUsername string, limit, offset int) ([]models.BidReview, error)
}

// PostgresBidRepository - реализация BidRepository для базы данных.
type PostgresBidRepository struct {
	DB *pgxpool.Pool
}

// NewPostgresBidRepository создает новый экземпляр PostgresBidRepository.
func NewPostgresBidRepository(db *pgxpool.Pool) *PostgresBidRepository {
	return &PostgresBidRepository{DB: db}
}

// CreateBid создает новое предложение.
func (r *PostgresBidRepository) CreateBid(ctx context.Context, bidReq models.BidRequest) (*models.Bid, error) {
	newBid := models.Bid{
		ID:          uuid.New().String(),
		Name:        bidReq.Name,
		Description: bidReq.Description,
		Status:      models.CreatedBid,
		TenderId:    bidReq.TenderId,
		AuthorType:  bidReq.AuthorType,
		AuthorId:    bidReq.AuthorId,
		Version:     1,
		CreatedAt:   time.Now().UTC(),
	}
	insertQuery := `INSERT INTO bid (id, name, description, status, tender_id, author_type, author_id, version, created_at)
                   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.DB.Exec(
		ctx,
		insertQuery,
		newBid.ID,
		newBid.Name,
		newBid.Description,
		newBid.Status,
		newBid.TenderId,
		newBid.AuthorType,
		newBid.AuthorId,
		newBid.Version,
		newBid.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &newBid, nil
}

// GetUserBid возвращает список предложений пользователя.
func (r *PostgresBidRepository) GetUserBid(ctx context.Context, limit, offset int, username string) ([]models.Bid, error) {
	var query string
	var args []interface{}
	if username != "" {
		query = `
			SELECT bid.id, bid.name, bid.description, bid.status, bid.tender_id, bid.author_type, bid.author_id, bid.version, bid.created_at
			FROM bid
			JOIN employee e ON bid.author_id = e.id
   			WHERE $1 = e.username
			ORDER BY bid.name
			LIMIT $2 OFFSET $3;`
		args = append(args, username, limit, offset)
	} else {
		query = `
			SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
			FROM bid
			ORDER BY name
			LIMIT $1 OFFSET $2`
		args = append(args, limit, offset)
	}

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userBids []models.Bid
	for rows.Next() {
		var bid models.Bid
		if err := rows.Scan(
			&bid.ID,
			&bid.Name,
			&bid.Description,
			&bid.Status,
			&bid.TenderId,
			&bid.AuthorType,
			&bid.AuthorId,
			&bid.Version,
			&bid.CreatedAt); err != nil {
			return nil, err
		}
		userBids = append(userBids, bid)
	}
	return userBids, nil
}

// GetTenderBid возвращает список предложений для тендера.
func (r *PostgresBidRepository) GetTenderBid(ctx context.Context, tenderId string, limit, offset int) ([]models.Bid, error) {
	query := `
		SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
		FROM bid
		WHERE tender_id = $1
		ORDER BY name
		LIMIT $2 OFFSET $3`
	rows, err := r.DB.Query(ctx, query, tenderId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []models.Bid
	for rows.Next() {
		var bid models.Bid
		if err := rows.Scan(&bid.ID, &bid.Name, &bid.Description, &bid.Status, &bid.TenderId, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt); err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}
	return bids, nil
}

// GetBidStatus возвращает статус предложения.
func (r *PostgresBidRepository) GetBidStatus(ctx context.Context, bidId string) (*models.BidStatus, error) {
	var status models.BidStatus
	query := `SELECT status FROM bid WHERE id = $1`
	err := r.DB.QueryRow(ctx, query, bidId).Scan(&status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// UpdateBidStatus меняет статус предложения.
func (r *PostgresBidRepository) UpdateBidStatus(ctx context.Context, bidId, status string) (*models.Bid, error) {
	updateQuery := `UPDATE bid SET status = $1 WHERE id = $2`
	_, err := r.DB.Exec(ctx, updateQuery, status, bidId)
	if err != nil {
		return nil, err
	}
	var bid models.Bid
	bidQuery := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
	             FROM bid WHERE id = $1`
	err = r.DB.QueryRow(ctx, bidQuery, bidId).Scan(
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

// EditBid меняет описание предложения.
func (r *PostgresBidRepository) EditBid(ctx context.Context, bidId string, updateFields map[string]interface{}) (*models.Bid, error) {
	var currentBid models.Bid
	selectQuery := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
	                FROM bid WHERE id = $1`
	err := r.DB.QueryRow(ctx, selectQuery, bidId).Scan(
		&currentBid.ID,
		&currentBid.Name,
		&currentBid.Description,
		&currentBid.Status,
		&currentBid.TenderId,
		&currentBid.AuthorType,
		&currentBid.AuthorId,
		&currentBid.Version,
		&currentBid.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	var bid models.Bid
	bidQuery := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
	             FROM bid WHERE id = $1`
	err = r.DB.QueryRow(ctx, bidQuery, bidId).Scan(
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

	var maxVersion int
	versionQuery := `SELECT COALESCE(MAX(version), 0) FROM bid_history WHERE bid_id = $1`
	err = r.DB.QueryRow(ctx, versionQuery, currentBid.ID).Scan(&maxVersion)
	if err != nil {
		return nil, err
	}

	historyInsertQuery := `INSERT INTO bid_history (bid_id, name, description, status, author_type, author_id, version, created_at)
                          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = r.DB.Exec(
		ctx,
		historyInsertQuery,
		currentBid.ID,
		currentBid.Name,
		currentBid.Description,
		currentBid.Status,
		currentBid.AuthorType,
		currentBid.AuthorId,
		maxVersion+1,
		currentBid.CreatedAt)
	if err != nil {
		return nil, err
	}

	var updates []string
	args := []interface{}{bidId} // Первый аргумент всегда будет bidId
	argIndex := 2

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

	if len(updates) == 0 {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "no valid fields to update")
	}

	updates = append(updates, "version = version + 1")
	updateQuery := fmt.Sprintf("UPDATE bid SET %s WHERE id = $1 RETURNING id, name, description, status, tender_id, author_type, author_id, version, created_at", strings.Join(updates, ", "))

	var updatedBid models.Bid
	err = r.DB.QueryRow(ctx, updateQuery, args...).Scan(
		&updatedBid.ID,
		&updatedBid.Name,
		&updatedBid.Description,
		&updatedBid.Status,
		&updatedBid.TenderId,
		&updatedBid.AuthorType,
		&updatedBid.AuthorId,
		&updatedBid.Version,
		&updatedBid.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &updatedBid, nil
}

// SubmitBidDecision отправляет решение по предложению.
func (r *PostgresBidRepository) SubmitBidDecision(ctx context.Context, bidId, decision string) (*models.Bid, error) {
	var bid models.Bid
	bidQuery := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
	             FROM bid WHERE id = $1`
	err := r.DB.QueryRow(ctx, bidQuery, bidId).Scan(
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
	bid.Status = models.BidStatus(decision)

	updateBidQuery := `UPDATE bid SET status = $1 WHERE id = $2`
	_, err = r.DB.Exec(ctx, updateBidQuery, bid.Status, bidId)
	if err != nil {
		return nil, err
	}

	if models.BidDecision(decision) == models.ApprovedBid {
		updateTenderQuery := `UPDATE tender SET status = $1 WHERE id = $2`
		_, err = r.DB.Exec(ctx, updateTenderQuery, models.ClosedTender, bid.TenderId)
		if err != nil {
			return nil, err
		}
	}

	return &bid, nil
}

// SubmitBidFeedback отправляет отзыв на предложение.
func (r *PostgresBidRepository) SubmitBidFeedback(ctx context.Context, review models.BidReview, bidId string) (*models.Bid, error) {
	insertQuery := `INSERT INTO bid_review (id, bid_id, description, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.DB.Exec(ctx, insertQuery, review.ID, review.BidID, review.Description, review.CreatedAt)
	if err != nil {
		return nil, err
	}

	var bid models.Bid
	selectQuery := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
                   FROM bid WHERE id = $1`
	err = r.DB.QueryRow(ctx, selectQuery, bidId).Scan(
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

// RollbackBid откатывает версию предложения.
func (r *PostgresBidRepository) RollbackBid(ctx context.Context, bidId string, version int) (*models.Bid, error) {
	var bid models.Bid
	bidQuery := `SELECT id, name, description, status, tender_id, author_type, author_id, version, created_at
	             FROM bid WHERE id = $1`
	err := r.DB.QueryRow(ctx, bidQuery, bidId).Scan(
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

	var rollbackBid models.Bid
	query := `SELECT bid_id, name, description, status, author_type, author_id, version, created_at
	          FROM bid_history WHERE bid_id = $1 AND version = $2`
	err = r.DB.QueryRow(ctx, query, bidId, version).Scan(
		&rollbackBid.ID,
		&rollbackBid.Name,
		&rollbackBid.Description,
		&rollbackBid.Status,
		&rollbackBid.AuthorType,
		&rollbackBid.AuthorId,
		&rollbackBid.Version,
		&rollbackBid.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	updateQuery := `
			UPDATE bid SET name = $1, description = $2, status = $3, author_type = $4, author_id = $5, version = version + 1
			WHERE id = $6 RETURNING id, name, description, status, author_type, author_id, version, created_at`
	err = r.DB.QueryRow(
		ctx,
		updateQuery,
		rollbackBid.Name,
		rollbackBid.Description,
		rollbackBid.Status,
		rollbackBid.AuthorType,
		rollbackBid.AuthorId,
		bidId).Scan(
		&bid.ID,
		&bid.Name,
		&bid.Description,
		&bid.Status,
		&bid.AuthorType,
		&bid.AuthorId,
		&bid.Version,
		&bid.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	historyInsertQuery := `INSERT INTO bid_history (bid_id, name, description, status, author_type, author_id, version, created_at)
                          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = r.DB.Exec(
		ctx,
		historyInsertQuery,
		bid.ID,
		bid.Name,
		bid.Description,
		bid.Status,
		bid.AuthorType,
		bid.AuthorId,
		bid.Version,
		bid.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &bid, nil
}

// GetBidReviews получает список отзывов на предложение.
func (r *PostgresBidRepository) GetBidReviews(ctx context.Context, tenderId, authorUsername, requesterUsername string, limit, offset int) ([]models.BidReview, error) {
	query := `
   	SELECT br.id, br.description, br.created_at
		FROM bid_review br
		JOIN bid b ON br.bid_id = b.id
		JOIN tender t ON b.tender_id = t.id
		WHERE t.id = $1
		AND b.author_id = (SELECT id FROM employee WHERE username = $2)
		AND EXISTS (
   		SELECT 1
   		FROM organization_responsible o
   		WHERE o.user_id = (SELECT id FROM employee WHERE username = $3)
   		AND o.organization_id = t.organization_id
		)
		ORDER BY br.created_at DESC
		LIMIT $4 OFFSET $5;
	`

	rows, err := r.DB.Query(ctx, query, tenderId, authorUsername, requesterUsername, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []models.BidReview
	for rows.Next() {
		var review models.BidReview
		if err := rows.Scan(&review.ID, &review.Description, &review.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}
