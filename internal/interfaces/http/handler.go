// Package http exposes the payment use cases as a REST API.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"payment-processing-platform/internal/application"
	"payment-processing-platform/internal/domain"
)

type Handler struct {
	create *application.CreatePayment
	get    *application.GetPayment
}

func NewHandler(create *application.CreatePayment, get *application.GetPayment) http.Handler {
	h := &Handler{create: create, get: get}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /payments", h.createPayment)
	mux.HandleFunc("GET /payments/{id}", h.getPayment)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return mux
}

type createRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type paymentResponse struct {
	ID        string    `json:"id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(p *domain.Payment) paymentResponse {
	return paymentResponse{p.ID, p.AmountCents, p.Currency, string(p.Status), p.CreatedAt, p.UpdatedAt}
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req createRequest
	if err := dec.Decode(&req); err != nil || dec.Decode(&struct{}{}) != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_body", "body must be a single JSON object with amount (integer cents) and currency")
		return
	}
	p, created, err := h.create.Execute(r.Context(), application.CreatePaymentInput{
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		AmountCents:    req.Amount,
		Currency:       req.Currency,
	})
	if err != nil {
		handleError(w, err)
		return
	}
	status := http.StatusOK // replay of an earlier request
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, toResponse(p))
}

func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a UUID")
		return
	}
	p, err := h.get.Execute(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(p))
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidIdempotencyKey):
		writeError(w, http.StatusBadRequest, "invalid_idempotency_key", err.Error())
	case errors.Is(err, domain.ErrInvalidAmount), errors.Is(err, domain.ErrInvalidCurrency):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, application.ErrIdempotencyConflict):
		writeError(w, http.StatusUnprocessableEntity, "idempotency_conflict", err.Error())
	case errors.Is(err, application.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		slog.Error("unexpected error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
