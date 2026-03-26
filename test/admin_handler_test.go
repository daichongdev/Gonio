package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goflow/internal/handler"
	"goflow/internal/pkg/errcode"
	resppkg "goflow/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type mockAdminLoginService struct {
	result   *resppkg.LoginResult
	err      error
	username string
	password string
}

func (m *mockAdminLoginService) Login(_ context.Context, username, password string) (*resppkg.LoginResult, error) {
	m.username = username
	m.password = password
	return m.result, m.err
}

func TestAdminHandlerLoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAdminLoginService{
		result: &resppkg.LoginResult{
			Token:    "token",
			ExpireAt: 100,
			User:     resppkg.AdminInfo{ID: 1, Username: "admin"},
		},
	}
	h := handler.NewAdminHandler(mockSvc)

	r := gin.New()
	r.POST("/admin/v1/login", h.Login)

	reqBody := []byte(`{"username":"admin","password":"123456"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if mockSvc.username != "admin" || mockSvc.password != "123456" {
		t.Fatalf("service input not match, got username=%s password=%s", mockSvc.username, mockSvc.password)
	}

	var resp resppkg.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected business code 0, got %d", resp.Code)
	}
}

func TestAdminHandlerLoginServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAdminLoginService{
		err: errcode.ErrAdminOrPassword(),
	}
	h := handler.NewAdminHandler(mockSvc)

	r := gin.New()
	r.POST("/admin/v1/login", h.Login)

	reqBody := []byte(`{"username":"admin","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var resp resppkg.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Code != errcode.CodeAdminOrPassword {
		t.Fatalf("expected business code %d, got %d", errcode.CodeAdminOrPassword, resp.Code)
	}
}
