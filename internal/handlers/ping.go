package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/senyabanana/tender-service/internal/utils"
)

// PingHandler обрабатывает GET запрос к /api/ping
func PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendErrorResponse(w, http.StatusBadRequest, "invalid method, only GET is allowed")
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "ok"); err != nil {
		log.Println(err)
	}
}
