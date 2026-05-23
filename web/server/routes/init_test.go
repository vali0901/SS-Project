package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"mqtt-streaming-server/utils"
)

//
// -------------------- handleBrokerInfo --------------------
//

func TestHandleBrokerInfo_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/broker-info", nil)
	rr := httptest.NewRecorder()

	handleBrokerInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json got %s", rr.Header().Get("Content-Type"))
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["port"] != "8883" {
		t.Fatalf("expected port 8883 got %s", resp["port"])
	}

	if resp["ip"] == "" {
		t.Fatal("expected non-empty ip")
	}
}

func TestHandleBrokerInfo_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/broker-info", nil)
	rr := httptest.NewRecorder()

	handleBrokerInfo(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Method not allowed") {
		t.Fatalf("expected error message got %s", rr.Body.String())
	}
}

//
// -------------------- getOutboundIP --------------------
//

func TestGetOutboundIP_FromEnvIP(t *testing.T) {
	os.Setenv("MQTT_HOST_IP", "127.0.0.1")
	defer os.Unsetenv("MQTT_HOST_IP")

	ip := getOutboundIP()

	if ip != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1 got %s", ip)
	}
}

//
// -------------------- withCORS --------------------
//

func TestWithCORS_GET(t *testing.T) {
	called := false

	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected handler to be called")
	}

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestWithCORS_OPTIONS(t *testing.T) {
	called := false

	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Fatal("next handler should NOT be called")
	}

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", rr.Code)
	}
}

//
// -------------------- withAuth --------------------
//

func TestWithAuth_MissingHeader(t *testing.T) {
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestWithAuth_BadFormat(t *testing.T) {
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "BadToken")

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestWithAuth_InvalidToken(t *testing.T) {
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestWithAuth_ValidToken(t *testing.T) {
	token, err := utils.GenerateToken("test@test.com", "user")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value("email")
		role := r.Context().Value("role")

		if email != "test@test.com" {
			t.Fatalf("expected email test@test.com got %v", email)
		}

		if role != "user" {
			t.Fatalf("expected role user got %v", role)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}

func TestWithAuth_ContextPropagation(t *testing.T) {
	token, err := utils.GenerateToken("admin@test.com", "admin")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "extra", "ok")

		if ctx.Value("extra") != "ok" {
			t.Fatal("context propagation failed")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
}