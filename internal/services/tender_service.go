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

type TenderService struct {
	Repo   repository.TenderRepository
	dbPool *pgxpool.Pool
}

// NewTenderService создаёт новый экземпляр TenderService.
func NewTenderService(repo repository.TenderRepository, dbPool *pgxpool.Pool) *TenderService {
	return &TenderService{Repo: repo, dbPool: dbPool}
}

// FetchTenders получает список тендеров.
func (s *TenderService) FetchTenders(ctx context.Context, limit, offset int, serviceTypes []string) ([]models.Tender, error) {
	allowedServiceTypes := map[models.TenderServiceType]bool{
		models.Construction: true,
		models.Delivery:     true,
		models.Manufacture:  true,
	}
	for _, serviceType := range serviceTypes {
		tenderServiceType := models.TenderServiceType(serviceType)
		if !allowedServiceTypes[tenderServiceType] {
			return nil, models.NewErrorResponse(http.StatusBadRequest, fmt.Sprintf("unsupported service type: %s", serviceType))
		}
	}
	return s.Repo.GetTenders(ctx, limit, offset, serviceTypes)
}

// CreateTender создает новый тендер.
func (s *TenderService) CreateTender(ctx context.Context, tenderReq models.TenderRequest) (*models.Tender, error) {
	if tenderReq.Name == "" || tenderReq.Description == "" || tenderReq.OrganizationID == "" || tenderReq.CreatorUsername == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required fields")
	}

	exists, err := utils.CheckUserExists(ctx, s.dbPool, tenderReq.CreatorUsername)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
	}
	if !exists {
		return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
	}

	isResponsible, err := utils.CheckUserResponsibleForOrganization(ctx, s.dbPool, tenderReq.CreatorUsername, tenderReq.OrganizationID)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
	}
	if !isResponsible {
		return nil, models.NewErrorResponse(http.StatusForbidden, "you are not authorized to create tenders for this organization")
	}

	allowedServiceTypes := map[models.TenderServiceType]bool{
		models.Construction: true,
		models.Delivery:     true,
		models.Manufacture:  true,
	}
	if !allowedServiceTypes[tenderReq.ServiceType] {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid service type")
	}

	return s.Repo.CreateTender(ctx, tenderReq)
}

// GetUserTender получает список тендеров для пользователя.
func (s *TenderService) GetUserTender(ctx context.Context, limitStr, offsetStr, username string) ([]models.Tender, error) {
	if username != "" {
		exists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
		}
		if !exists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
	}

	limit, offset, err := utils.ParseLimitOffset(limitStr, offsetStr)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusBadRequest, err.Error())
	}
	return s.Repo.GetUserTender(ctx, limit, offset, username)
}

// GetTenderStatus получает статус тендера.
func (s *TenderService) GetTenderStatus(ctx context.Context, tenderId, username string) (models.TenderStatus, error) {
	if username != "" {
		exists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return "", models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
		}
		if !exists {
			return "", models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
		isAuthorized, err := utils.CheckUserAuthorized(ctx, s.dbPool, username, tenderId)
		if err != nil {
			return "", models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
		}
		if !isAuthorized {
			return "", models.NewErrorResponse(http.StatusForbidden, "you are not authorized to edit this tender")
		}
	}
	tenderExists, err := utils.CheckTenderExists(ctx, s.dbPool, tenderId)
	if !tenderExists && err != nil {
		return "", models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}
	return s.Repo.GetTenderStatus(ctx, tenderId, username)
}

// UpdateTenderStatus меняет статус тендера.
func (s *TenderService) UpdateTenderStatus(ctx context.Context, tenderId, status, username string) (*models.Tender, error) {
	if status == "" || username == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameters: status or username")
	}
	if username != "" {
		exists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
		}
		if !exists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
		isAuthorized, err := utils.CheckUserAuthorized(ctx, s.dbPool, username, tenderId)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server error")
		}
		if !isAuthorized {
			return nil, models.NewErrorResponse(http.StatusForbidden, "you are not authorized to edit this tender")
		}
	}

	currentTender, err := utils.GetTenderById(ctx, s.dbPool, tenderId)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}

	allowedStatusTransition := map[models.TenderStatus][]models.TenderStatus{
		models.CreatedTender:   {models.PublishedTender, models.ClosedTender},
		models.PublishedTender: {models.ClosedTender},
		models.ClosedTender:    {},
	}

	validTransition := allowedStatusTransition[currentTender.Status]
	if !utils.ContainsTender(validTransition, models.TenderStatus(status)) {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid tender status")
	}
	return s.Repo.UpdateTenderStatus(ctx, tenderId, status)
}

// EditTender меняет описание тендера.
func (s *TenderService) EditTender(ctx context.Context, tenderId, username string, updateFields map[string]interface{}) (*models.Tender, error) {
	if username == "" || tenderId == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameter: tenderId or username")
	}

	tenderExists, err := utils.CheckTenderExists(ctx, s.dbPool, tenderId)
	if !tenderExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}

	if username != "" {
		exists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server 1error")
		}
		if !exists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
		isAuthorized, err := utils.CheckUserAuthorized(ctx, s.dbPool, username, tenderId)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server 2error")
		}
		if !isAuthorized {
			return nil, models.NewErrorResponse(http.StatusForbidden, "you are not authorized to edit this tender")
		}
	}
	return s.Repo.EditTender(ctx, tenderId, updateFields)
}

// RollbackTender откатывает версию тендера
func (s *TenderService) RollbackTender(ctx context.Context, tenderId, username, versionStr string) (*models.Tender, error) {
	if username == "" || tenderId == "" || versionStr == "" {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "missing required query parameter: tenderId or username or version")
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, models.NewErrorResponse(http.StatusBadRequest, "invalid version number")
	}

	tenderExists, err := utils.CheckTenderExists(ctx, s.dbPool, tenderId)
	if !tenderExists && err != nil {
		return nil, models.NewErrorResponse(http.StatusNotFound, "tender not found")
	}

	if username != "" {
		exists, err := utils.CheckUserExists(ctx, s.dbPool, username)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server 1error")
		}
		if !exists {
			return nil, models.NewErrorResponse(http.StatusUnauthorized, "user does not exist")
		}
		isAuthorized, err := utils.CheckUserAuthorized(ctx, s.dbPool, username, tenderId)
		if err != nil {
			return nil, models.NewErrorResponse(http.StatusInternalServerError, "internal server 2error")
		}
		if !isAuthorized {
			return nil, models.NewErrorResponse(http.StatusForbidden, "you are not authorized to edit this tender")
		}
	}
	return s.Repo.RollbackTender(ctx, tenderId, version)
}
