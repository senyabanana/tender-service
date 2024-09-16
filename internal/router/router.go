package router

import (
	"net/http"

	"github.com/senyabanana/tender-service/internal/handlers"
)

func InitRoutes(tenderHandler *handlers.TenderHandler, bidHandler *handlers.BidHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/ping", handlers.PingHandler)
	mux.HandleFunc("/api/tenders", tenderHandler.GetTenders)
	mux.HandleFunc("/api/tenders/new", tenderHandler.CreateTender)
	mux.HandleFunc("/api/tenders/my", tenderHandler.GetUserTender)
	mux.HandleFunc("GET /api/tenders/{tenderId}/status", tenderHandler.GetTenderStatus)
	mux.HandleFunc("PUT /api/tenders/{tenderId}/status", tenderHandler.UpdateTenderStatus)
	mux.HandleFunc("/api/tenders/{tenderId}/edit", tenderHandler.EditTender)
	mux.HandleFunc("/api/tenders/{tenderId}/rollback/{version}", tenderHandler.RollbackTender)

	mux.HandleFunc("/api/bids/new", bidHandler.CreateBid)
	mux.HandleFunc("/api/bids/my", bidHandler.GetUserBid)
	mux.HandleFunc("/api/bids/{tenderId}/list", bidHandler.GetTenderBid)
	mux.HandleFunc("GET /api/bids/{bidId}/status", bidHandler.GetBidStatus)
	mux.HandleFunc("PUT /api/bids/{bidId}/status", bidHandler.UpdateBidStatus)
	mux.HandleFunc("/api/bids/{bidId}/edit", bidHandler.EditBid)
	mux.HandleFunc("/api/bids/{bidId}/submit_decision", bidHandler.SubmitBidDecision)
	mux.HandleFunc("/api/bids/{bidId}/feedback", bidHandler.SubmitBidFeedback)
	mux.HandleFunc("/api/bids/{bidId}/rollback/{version}", bidHandler.RollbackBid)
	mux.HandleFunc("/api/bids/{tenderId}/reviews", bidHandler.GetBidReviews)

	return mux
}
