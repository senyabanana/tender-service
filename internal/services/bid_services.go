package services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/senyabanana/tender-service/internal/models"
	"github.com/senyabanana/tender-service/internal/repository"
	"github.com/senyabanana/tender-service/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BidService struct {
	Repo   repository.BidRepository
	dbPool *pgxpool.Pool
}

// NewBidService создает новый экземпляр BidService.
func NewBidService(repo repository.BidRepository, dbPool *pgxpool.Pool) *BidService {
	return &BidService{Repo: repo, dbPool: dbPool}
}

// CreateBid создает новое предложение.
func (s *BidService) CreateBid(ctx context.Context, bidReq models.BidRequest) (*models.Bid, error) {
	if bidReq.Name == "" || bidReq.Description == "" || bidReq.TenderId == "" || bidReq.AuthorType == "" || bidReq.AuthorId == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required fields")
	}

	if bidReq.AuthorType != models.User && bidReq.AuthorType != models.Organization {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid author type. Must be 'Organization' or 'User'")
	}

	if bidReq.AuthorType == models.Organization {
		orgExists, err := utils.CheckUserInAnyOrganization(ctx, s.dbPool, bidReq.AuthorId)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
		}
		if !orgExists {
			return nil, models.NewErrorResponse(http.StatusForbidden, fmt.Sprintf("%s is not in any organizations", bidReq.AuthorId))
		}
	} else if bidReq.AuthorType == models.User {
		userExists, err := utils.CheckUserExistsById(ctx, s.dbPool, bidReq.AuthorId)
		if !userExists || err != nil {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
	}

	tenderExists, err := utils.CheckTenderExists(ctx, s.dbPool, bidReq.TenderId)
	if !tenderExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}
	return s.Repo.CreateBid(ctx, bidReq)
}

// GetUserBid получает список предложений для пользователя.
func (s *BidService) GetUserBid(ctx context.Context, limitStr, offsetStr, username string) ([]models.Bid, error) {
	limit, offset, err := utils.ParseLimitOffset(limitStr, offsetStr)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusBadRequest, err.Error())
	}

	if username != "" {
		userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
		}
		if !userExists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
	}
	return s.Repo.GetUserBid(ctx, limit, offset, username)
}

// GetTenderBid получает список предложений для тендера.
func (s *BidService) GetTenderBid(ctx context.Context, username, tenderId, limitStr, offsetStr string) ([]models.Bid, error) {
	limit, offset, err := utils.ParseLimitOffset(limitStr, offsetStr)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusBadRequest, err.Error())
	}

	if tenderId == "" || username == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameters: username or tenderId")
	}

	userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
	}
	if !userExists {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist or is incorrect")
	}

	tenderExists, err := utils.CheckTenderExists(ctx, s.dbPool, tenderId)
	if !tenderExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}

	isAuthorized, err := utils.CheckUserAuthorized(ctx, s.dbPool, username, tenderId)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user authorization")
	}
	if !isAuthorized {
		return nil, models.NewErrorResponse(http.StatusForbidden, "user is not authorized to view bids for this tender")
	}
	return s.Repo.GetTenderBid(ctx, tenderId, limit, offset)
}

// GetBidStatus получает статут предложения.
func (s *BidService) GetBidStatus(ctx context.Context, bidId, username string) (*models.BidStatus, error) {
	if username == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "username is required")
	}

	userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
	}
	if !userExists {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
	}
	return s.Repo.GetBidStatus(ctx, bidId)
}

// UpdateBidStatus меняет статус предложения.
func (s *BidService) UpdateBidStatus(ctx context.Context, bidId, status, username string) (*models.Bid, error) {
	if status == "" || username == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameters: username or status")
	}

	if username != "" {
		userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
		}
		if !userExists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
		isAuthorized, err := utils.CheckUserAuthorizedForBid(ctx, s.dbPool, username, bidId)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user authorization")
		}
		if !isAuthorized {
			return nil, models.NewErrorResponse(http.StatusForbidden, "user is not authorized to view bids for this tender")
		}
	}

	currentBid, err := utils.GetBidById(ctx, s.dbPool, bidId)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "bid not found")
	}

	allowedStatusTransition := map[models.BidStatus][]models.BidStatus{
		models.CreatedBid:   {models.PublishedBid, models.CanceledBid},
		models.PublishedBid: {models.CanceledBid},
		models.CanceledBid:  {},
	}

	validTransition := allowedStatusTransition[currentBid.Status]
	if !utils.ContainsBid(validTransition, models.BidStatus(status)) {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid bid status")
	}
	return s.Repo.UpdateBidStatus(ctx, bidId, status)
}

// EditBid меняет описание предложения.
func (s *BidService) EditBid(ctx context.Context, bidId, username string, updateFields map[string]interface{}) (*models.Bid, error) {
	if username == "" || bidId == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameter: bidId or username")
	}

	bidExists, err := utils.CheckBidExists(ctx, s.dbPool, bidId)
	if !bidExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "bid not found")
	}

	if username != "" {
		userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
		}
		if !userExists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
	}
	return s.Repo.EditBid(ctx, bidId, updateFields)
}

// SubmitBidDecision отправляет решение по предложению.
func (s *BidService) SubmitBidDecision(ctx context.Context, bidId, username, decision string) (*models.Bid, error) {
	if decision == "" || username == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "decision and username are required")
	}

	allowedDecision := map[models.BidDecision]bool{
		models.ApprovedBid: true,
		models.RejectedBid: true,
	}
	if !allowedDecision[models.BidDecision(decision)] {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid decision, must be either 'Approved' or 'Rejected'")
	}

	userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
	}
	if !userExists {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
	}
	return s.Repo.SubmitBidDecision(ctx, bidId, decision)
}

// SubmitBidFeedback отправляет отзыв на предложение.
func (s *BidService) SubmitBidFeedback(ctx context.Context, review models.BidReview, bidId, bidFeedback, username string) (*models.Bid, error) {
	if bidFeedback == "" || username == "" || bidId == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "bidFeedback, username and bidId are required")
	}

	bidExists, err := utils.CheckBidExists(ctx, s.dbPool, bidId)
	if !bidExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "bid not found")
	}

	userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
	}
	if !userExists {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
	}

	return s.Repo.SubmitBidFeedback(ctx, review, bidId)
}

// RollbackBid откатывает версию тендера.
func (s *BidService) RollbackBid(ctx context.Context, bidId, username, versionStr string) (*models.Bid, error) {
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid version number")
	}
	bidExists, err := utils.CheckBidExists(ctx, s.dbPool, bidId)
	if !bidExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "bid not found")
	}

	if username == "" || bidId == "" || versionStr == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameter: bidId or username or version")
	}

	userExists, err := utils.CheckUserExists(ctx, s.dbPool, username)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "failed to check user existence")
	}
	if !userExists {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
	}
	return s.Repo.RollbackBid(ctx, bidId, version)
}

// GetBidReviews получает список отзывов на предложение.
func (s *BidService) GetBidReviews(ctx context.Context, tenderId, authorUsername, requesterUsername, limitStr, offsetStr string) ([]models.BidReview, error) {
	limit, offset, err := utils.ParseLimitOffset(limitStr, offsetStr)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusBadRequest, err.Error())
	}

	if tenderId == "" || authorUsername == "" || requesterUsername == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameters: tenderId, authorUsername or requesterUsername")
	}

	tenderExists, err := utils.CheckTenderExists(ctx, s.dbPool, tenderId)
	if !tenderExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}

	requesterExists, err := utils.CheckUserExists(ctx, s.dbPool, requesterUsername)
	if !requesterExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "requester does not exist")
	}

	authorExists, err := utils.CheckUserExists(ctx, s.dbPool, authorUsername)
	if !authorExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "author does not exist")
	}

	isAuthorized, err := utils.CheckUserAuthorized(ctx, s.dbPool, requesterUsername, tenderId)
	if !isAuthorized && err != nil {
		return nil, models.NewErrorResponse(http.StatusForbidden, "user is not authorized to view bid reviews for this tender")
	}

	return s.Repo.GetBidReviews(ctx, tenderId, authorUsername, requesterUsername, limit, offset)
}
