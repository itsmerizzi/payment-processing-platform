package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"payment-processing-platform/internal/application"
	"payment-processing-platform/internal/application/fakes"
	apihttp "payment-processing-platform/internal/interfaces/http"
)

func newServer() http.Handler {
	repo := fakes.NewRepo()
	clock := fakes.FixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return apihttp.NewHandler(application.NewCreatePayment(repo, &fakes.SeqIDs{}, clock), application.NewGetPayment(repo))
}

func do(h http.Handler, method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreatePayment(t *testing.T) {
	h := newServer()
	body := `{"amount": 1000, "currency": "BRL"}`

	first := do(h, "POST", "/payments", "k1", body)
	if first.Code != 201 {
		t.Fatalf("first = %d %s", first.Code, first.Body)
	}
	var p map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p["status"] != "PENDING" || p["amount"] != float64(1000) || p["currency"] != "BRL" || p["id"] == "" {
		t.Fatalf("body = %v", p)
	}

	replay := do(h, "POST", "/payments", "k1", body)
	if replay.Code != 200 || replay.Body.String() != first.Body.String() {
		t.Fatalf("replay = %d %s", replay.Code, replay.Body)
	}

	conflict := do(h, "POST", "/payments", "k1", `{"amount": 999, "currency": "BRL"}`)
	if conflict.Code != 422 || !strings.Contains(conflict.Body.String(), "idempotency_conflict") {
		t.Fatalf("conflict = %d %s", conflict.Code, conflict.Body)
	}
}

func TestCreatePaymentValidation(t *testing.T) {
	h := newServer()
	tests := []struct {
		name, key, body, code string
	}{
		{"missing key", "", `{"amount":1,"currency":"BRL"}`, "invalid_idempotency_key"},
		{"long key", strings.Repeat("a", 256), `{"amount":1,"currency":"BRL"}`, "invalid_idempotency_key"},
		{"malformed json", "k", `{`, "invalid_body"},
		{"unknown field", "k", `{"amount":1,"currency":"BRL","x":1}`, "invalid_body"},
		{"amount as string", "k", `{"amount":"1","currency":"BRL"}`, "invalid_body"},
		{"fractional amount", "k", `{"amount":1.5,"currency":"BRL"}`, "invalid_body"},
		{"trailing data", "k", `{"amount":1,"currency":"BRL"}{}`, "invalid_body"},
		{"zero amount", "k", `{"amount":0,"currency":"BRL"}`, "validation_error"},
		{"negative amount", "k", `{"amount":-1,"currency":"BRL"}`, "validation_error"},
		{"missing amount", "k", `{"currency":"BRL"}`, "validation_error"},
		{"bad currency", "k", `{"amount":1,"currency":"REAL"}`, "validation_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, "POST", "/payments", tt.key, tt.body)
			var out struct {
				Error struct{ Code string } `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 400 || out.Error.Code != tt.code {
				t.Fatalf("got %d %s, want 400 %s", rec.Code, rec.Body, tt.code)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("content-type = %q", ct)
			}
		})
	}
}

func TestGetPayment(t *testing.T) {
	h := newServer()
	// The fake ID generator is not a UUID, so seed through a real UUID-shaped lookup:
	// a valid UUID that does not exist is 404, anything else is 400.
	if rec := do(h, "GET", "/payments/0190a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b", "", ""); rec.Code != 404 {
		t.Fatalf("missing = %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "GET", "/payments/not-a-uuid", "", ""); rec.Code != 400 {
		t.Fatalf("invalid = %d %s", rec.Code, rec.Body)
	}
}

func TestGetExistingPayment(t *testing.T) {
	repo := fakes.NewRepo()
	clock := fakes.FixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	ids := uuidIDs{}
	h := apihttp.NewHandler(application.NewCreatePayment(repo, ids, clock), application.NewGetPayment(repo))

	var created struct{ ID string }
	rec := do(h, "POST", "/payments", "k", `{"amount":500,"currency":"usd"}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	got := do(h, "GET", "/payments/"+created.ID, "", "")
	if got.Code != 200 || got.Body.String() != rec.Body.String() {
		t.Fatalf("get = %d %s", got.Code, got.Body)
	}
}

type uuidIDs struct{}

func (uuidIDs) NewID() (string, error) { return "0190a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b", nil }
