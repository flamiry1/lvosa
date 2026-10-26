package handlers

import (
	"context"
	"errors"
	"lvosa/backend/internal/models"
	"lvosa/backend/internal/service"
	"lvosa/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	s := store.New()
	srv := service.New(s)
	r := gin.New()
	r.Use(gin.Recovery())
	h := New(srv)
	h.RegisterRouters(r)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		setCookie(t, token, req)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHandlers(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		seed       []string
		wantStatus int
	}{
		{name: "create", method: http.MethodPost, path: "/item", body: `{"name": "Kupijo mleko jutri"}`, wantStatus: http.StatusCreated},
		{name: "create double", method: http.MethodPost, path: "/item", body: `{"name": "Kuhinja"}`, seed: []string{"Kuhinja"}, wantStatus: http.StatusConflict},
		{name: "create empty name", method: http.MethodPost, path: "/item", body: `{"name": "  "}`, wantStatus: http.StatusBadRequest},
		{name: "create without name field", method: http.MethodPost, path: "/item", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "create with broken JSON", method: http.MethodPost, path: "/item", body: `{"name": "boss"`, wantStatus: http.StatusBadRequest},
		{name: "get created", method: http.MethodGet, path: "/item/1", body: "", seed: []string{"Kupijo mleko jutri"}, wantStatus: http.StatusOK},
		{name: "get all items", method: http.MethodGet, path: "/items", body: "", wantStatus: http.StatusOK},
		{name: "get unexisting", method: http.MethodGet, path: "/item/99", body: "", wantStatus: http.StatusNotFound},
		{name: "get invalid ID", method: http.MethodGet, path: "/item/abc", body: "", wantStatus: http.StatusBadRequest},
		{name: "updated existing", method: http.MethodPut, path: "/item/1", body: `{"name": "Kupi mleko"}`, seed: []string{"Kupijo mleko jutri"}, wantStatus: http.StatusOK},
		{name: "updated unexisting", method: http.MethodPut, path: "/item/99", body: `{"name": "Kupi mleko"}`, seed: []string{"Kupijo mleko jutri"}, wantStatus: http.StatusNotFound},
		{name: "updated with invalid ID", method: http.MethodPut, path: "/item/abc", body: `{"name": "Kupi mleko"}`, seed: []string{"Kupijo mleko jutri"}, wantStatus: http.StatusBadRequest},
		{name: "updated existing name", method: http.MethodPut, path: "/item/2", body: `{"name": "PRVI"}`, seed: []string{"prvi", "drugi"}, wantStatus: http.StatusConflict},
		{name: "delete exising", method: http.MethodDelete, path: "/item/1", body: "", seed: []string{"Sonce vstaja"}, wantStatus: http.StatusNoContent},
		{name: "delete unexisting", method: http.MethodDelete, path: "/item/99", body: "", seed: []string{"Sonce vstaja"}, wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newTestRouter(t)
			for _, item := range test.seed {
				rec := doRequest(t, r, http.MethodPost, "/item", `{"name": "`+item+`"}`, "")
				if rec.Code != http.StatusCreated {
					t.Fatalf("seed %q:, status = %d, want %d", item, rec.Code, http.StatusCreated)
				}
			}
			rec := doRequest(t, r, test.method, test.path, test.body, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("expected status = %d, received %d", test.wantStatus, rec.Code)
			}
		})
	}
}

func TestServiceHandlers(t *testing.T) {
	fake := &fakeService{err: errors.New("db down")}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	h := New(fake)
	h.RegisterRouters(r)
	tests := []struct {
		name        string
		path        string
		wantCode    int
		wantMessage string
	}{
		{name: "internal error trying to get one item", path: "/item/1", wantCode: http.StatusInternalServerError, wantMessage: `{"status":"internal server error"}`},
		{name: "internal error trying to get all items", path: "/items", wantCode: http.StatusInternalServerError, wantMessage: `{"status":"internal server error"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := doRequest(t, r, http.MethodGet, test.path, "", "")
			if rec.Code != test.wantCode {
				t.Fatalf("unexpected status = %d, body %q", rec.Code, rec.Body.String())
			}
			if rec.Body.String() != test.wantMessage {
				t.Fatalf("wrong internal error message %s, want %s", rec.Body.String(), test.wantMessage)
			}
		})
	}

}

type fakeService struct {
	err error
}

func (f *fakeService) List(ctx context.Context) ([]models.Item, error) {
	return []models.Item{}, f.err
}

func (f *fakeService) Get(ctx context.Context, id int) (models.Item, error) {
	return models.Item{}, f.err
}

func (f *fakeService) Create(ctx context.Context, item models.Item) (models.Item, error) {
	return models.Item{}, nil
}

func (f *fakeService) Update(ctx context.Context, item models.Item, id int) error {
	return nil
}

func (f *fakeService) Delete(ctx context.Context, id int) error {
	return nil
}
