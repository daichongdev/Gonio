package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gonio/internal/handler"
	"gonio/internal/model"
	"gonio/internal/pkg/errcode"
	resppkg "gonio/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type mockProductService struct {
	created    *model.Product
	createErr  error
	getProduct *model.Product
	getErr     error
}

func (m *mockProductService) List(_ context.Context, _ int, _ int) ([]model.Product, int64, error) {
	return nil, 0, nil
}

func (m *mockProductService) GetByID(_ context.Context, _ uint) (*model.Product, error) {
	return m.getProduct, m.getErr
}

func (m *mockProductService) Create(_ context.Context, product *model.Product) error {
	m.created = product
	return m.createErr
}

func (m *mockProductService) Update(_ context.Context, _ *model.Product) error {
	return nil
}

func (m *mockProductService) Delete(_ context.Context, _ uint) error {
	return nil
}

func TestProductHandlerCreateSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockProductService{}
	h := handler.NewProductHandler(mockSvc)

	r := gin.New()
	r.POST("/admin/v1/products", h.Create)

	reqBody := []byte(`{"name":"n1","description":"desc","price":9.9,"stock":3,"category_id":8}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/products", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if mockSvc.created == nil {
		t.Fatalf("expected service Create called")
	}

	var resp resppkg.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected business code 0, got %d", resp.Code)
	}
}

func TestProductHandlerGetSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	product := &model.Product{
		Name:        "test product",
		Description: "desc",
		Price:       9.9,
		Stock:       10,
		CategoryID:  1,
	}
	mockSvc := &mockProductService{getProduct: product}
	h := handler.NewProductHandler(mockSvc)

	r := gin.New()
	r.GET("/admin/v1/products/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/products/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp resppkg.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func TestProductHandlerGetNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockProductService{getErr: errcode.ErrProductNotFound()}
	h := handler.NewProductHandler(mockSvc)

	r := gin.New()
	r.GET("/admin/v1/products/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/products/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestProductHandlerGetInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockProductService{}
	h := handler.NewProductHandler(mockSvc)

	r := gin.New()
	r.GET("/admin/v1/products/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/products/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
