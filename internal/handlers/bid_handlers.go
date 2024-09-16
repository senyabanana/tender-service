package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/senyabanana/tender-service/internal/models"
	"github.com/senyabanana/tender-service/internal/services"
	"github.com/senyabanana/tender-service/internal/utils"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BidHandler - структура для обработки HTTP-запросов.
type BidHandler struct {
	Service *services.BidService
	Logger  *log.Logger
	Timeout time.Duration
	dbPool  *pgxpool.Pool
}

// NewBIdHandler создает новый экземпляр BidHandler.
func NewBIdHandler(service *services.BidService, logger *log.Logger, timeout time.Duration, dbPool *pgxpool.Pool) *BidHandler {
	return &BidHandler{
		Service: service,
		Logger:  logger,
		Timeout: timeout,
		dbPool:  dbPool,
	}
}

// CreateBid обрабатывает запросы для создания предложения.
func (h *BidHandler) CreateBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only POST is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	var bidReq models.BidRequest
	err := json.NewDecoder(r.Body).Decode(&bidReq)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newBid, err := h.Service.CreateBid(ctx, bidReq)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to create bid")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(newBid); err != nil {
		h.Logger.Println(err)
	}
}

// GetUserBid обрабатывает запросы для получения списка предложений пользователя.
func (h *BidHandler) GetUserBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	username := r.URL.Query().Get("username")

	userBids, err := h.Service.GetUserBid(ctx, limitStr, offsetStr, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to retrieve bids")
		return
	}

	if len(userBids) == 0 {
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusNotFound, "no bids found for this user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(userBids); err != nil {
		h.Logger.Println(err)
	}
}

// GetTenderBid обрабатывает запросы для получения списка предложений по тендеру.
func (h *BidHandler) GetTenderBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	tenderId := r.PathValue("tenderId")
	username := r.URL.Query().Get("username")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	bids, err := h.Service.GetTenderBid(ctx, username, tenderId, limitStr, offsetStr)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to retrieve bids for tender")
		return
	}

	if len(bids) == 0 {
		utils.SendErrorResponse(w, http.StatusNotFound, "no bids found for the specified tender")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(bids); err != nil {
		h.Logger.Println(err)
	}
}

// GetBidStatus обрабатывает запросы для получения статуса предложения.
func (h *BidHandler) GetBidStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	bidId := r.PathValue("bidId")
	username := r.URL.Query().Get("username")

	status, err := h.Service.GetBidStatus(ctx, bidId, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to check user authorization")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(status); err != nil {
		h.Logger.Println(err)
	}
}

// UpdateBidStatus обрабатывает запросы для изменения статуса предложения.
func (h *BidHandler) UpdateBidStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PATCH is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	bidId := r.PathValue("bidId")
	username := r.URL.Query().Get("username")
	status := r.URL.Query().Get("status")

	bid, err := h.Service.UpdateBidStatus(ctx, bidId, status, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to update bid status")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(bid); err != nil {
		h.Logger.Println(err)
	}
}

// EditBid обрабатывает запросы изменения предложения.
func (h *BidHandler) EditBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PATCH is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	bidId := r.PathValue("bidId")
	username := r.URL.Query().Get("username")

	var updateFields map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateFields); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedBid, err := h.Service.EditBid(ctx, bidId, username, updateFields)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to update bid")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(updatedBid); err != nil {
		h.Logger.Println(err)
	}
}

// SubmitBidDecision обрабатывает запросы по отправке решения по предложению.
func (h *BidHandler) SubmitBidDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PUT is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	bidId := r.PathValue("bidId")
	decision := r.URL.Query().Get("decision")
	username := r.URL.Query().Get("username")

	bid, err := h.Service.SubmitBidDecision(ctx, bidId, username, decision)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to update tender status")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(bid); err != nil {
		h.Logger.Println(err)
	}
}

// SubmitBidFeedback обрабатывает запросы на отправку отзыва на предложение.
func (h *BidHandler) SubmitBidFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PUT is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	bidId := r.PathValue("bidId")
	bidFeedback := r.URL.Query().Get("bidFeedback")
	username := r.URL.Query().Get("username")

	review := models.BidReview{
		ID:          uuid.New().String(),
		BidID:       bidId,
		Description: bidFeedback,
		CreatedAt:   time.Now().UTC(),
	}

	bid, err := h.Service.SubmitBidFeedback(ctx, review, bidId, bidFeedback, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to retrieve bid")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(bid); err != nil {
		h.Logger.Println(err)
	}
}

// RollbackBid обрабатывает запросы для отката версии предложения.
func (h *BidHandler) RollbackBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PUT is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	bidId := r.PathValue("bidId")
	versionStr := r.PathValue("version")
	username := r.URL.Query().Get("username")

	bid, err := h.Service.RollbackBid(ctx, bidId, username, versionStr)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to save bid history")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(bid); err != nil {
		h.Logger.Println(err)
	}
}

// GetBidReviews обрабатывает запросы на просмотр отзывов на предложение.
func (h *BidHandler) GetBidReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	tenderId := r.PathValue("tenderId")
	authorUsername := r.URL.Query().Get("authorUsername")
	requesterUsername := r.URL.Query().Get("requesterUsername")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	reviews, err := h.Service.GetBidReviews(ctx, tenderId, authorUsername, requesterUsername, limitStr, offsetStr)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to parse database response")
		return
	}

	if len(reviews) == 0 {
		utils.SendErrorResponse(w, http.StatusNotFound, fmt.Sprintf("%s has no bids or reviews on tender", authorUsername))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(reviews); err != nil {
		h.Logger.Println(err)
	}
}
