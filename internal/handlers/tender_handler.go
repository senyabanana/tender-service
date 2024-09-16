package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/senyabanana/tender-service/internal/models"
	"github.com/senyabanana/tender-service/internal/services"
	"github.com/senyabanana/tender-service/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TenderHandler - структура для обработки HTTP-запросов.
type TenderHandler struct {
	Service *services.TenderService
	Logger  *log.Logger
	Timeout time.Duration
	dbPool  *pgxpool.Pool
}

// NewTenderHandler создаёт новый экземпляр TenderHandler.
func NewTenderHandler(service *services.TenderService, logger *log.Logger, timeout time.Duration, dbPool *pgxpool.Pool) *TenderHandler {
	return &TenderHandler{
		Service: service,
		Logger:  logger,
		Timeout: timeout,
		dbPool:  dbPool,
	}
}

// GetTenders обрабатывает запросы для получения списка тендеров.
func (h *TenderHandler) GetTenders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	serviceTypes := r.URL.Query()["service_type"]

	limit, offset, err := utils.ParseLimitOffset(limitStr, offsetStr)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	tenders, err := h.Service.FetchTenders(ctx, limit, offset, serviceTypes)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to fetch tenders")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(tenders); err != nil {
		h.Logger.Println(err)
	}
}

// CreateTender обрабатывает запросы для создания тендера.
func (h *TenderHandler) CreateTender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only POST is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	var tenderReq models.TenderRequest
	err := json.NewDecoder(r.Body).Decode(&tenderReq)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tender, err := h.Service.CreateTender(ctx, tenderReq)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to create tender")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(tender); err != nil {
		h.Logger.Println(err)
	}
}

// GetUserTender обрабатывает запросы для получения списка тендеров пользователя.
func (h *TenderHandler) GetUserTender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	username := r.URL.Query().Get("username")

	tenders, err := h.Service.GetUserTender(ctx, limitStr, offsetStr, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println("Error fetching tenders:", err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to fetch tenders")
		return
	}

	if len(tenders) == 0 {
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusNotFound, "no tenders found for this user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(tenders); err != nil {
		h.Logger.Println(err)
	}
}

// GetTenderStatus обрабатывает запросы для получения статуса тендера.
func (h *TenderHandler) GetTenderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	tenderId := r.PathValue("tenderId")
	username := r.URL.Query().Get("username")

	status, err := h.Service.GetTenderStatus(ctx, tenderId, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to fetch tenders")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(status); err != nil {
		h.Logger.Println(err)
	}
}

// UpdateTenderStatus обрабатывает запросы для изменения статуса тендера.
func (h *TenderHandler) UpdateTenderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PUT is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	tenderId := r.PathValue("tenderId")
	status := r.URL.Query().Get("status")
	username := r.URL.Query().Get("username")

	tender, err := h.Service.UpdateTenderStatus(ctx, tenderId, status, username)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to fetch tenders")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(tender); err != nil {
		log.Println(err)
	}
}

// EditTender обрабатывает запросы для изменения тендера.
func (h *TenderHandler) EditTender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PATCH is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	tenderId := r.PathValue("tenderId")
	username := r.URL.Query().Get("username")

	var updateFields map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateFields); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedTender, err := h.Service.EditTender(ctx, tenderId, username, updateFields)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to update tender")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(updatedTender); err != nil {
		h.Logger.Println(err)
	}
}

// RollbackTender обрабатывает запросы для отката версии тендера.
func (h *TenderHandler) RollbackTender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only PUT is allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Timeout)
	defer cancel()

	tenderId := r.PathValue("tenderId")
	versionStr := r.PathValue("version")
	username := r.URL.Query().Get("username")

	updatedTender, err := h.Service.RollbackTender(ctx, tenderId, username, versionStr)
	if err != nil {
		if errorResponse, ok := err.(*models.ErrorResponse); ok {
			h.Logger.Println(err)
			utils.SendErrorResponse(w, errorResponse.StatusCode, errorResponse.Message)
			return
		}
		h.Logger.Println(err)
		utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to rollback tender")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(updatedTender); err != nil {
		h.Logger.Println(err)
	}
}
