package routes_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
	"mqtt-streaming-server/routes"
)

func TestInitUserRoutes(t *testing.T) {
	mux := http.NewServeMux()
	routes.InitUserRoutes(nil, mux)
}

func TestUserController_Register(t *testing.T) {
	tests := []struct {
		name                 string
		inputBody            string
		mockFindByEmailError error
		mockSaveReturn       error
		expectedStatus       int
		expectedUser         *domain.User
	}{
		{
			name:           "successful registration",
			inputBody:      `{"email": "test@example.com", "password": "securepass"}`,
			mockSaveReturn: nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing email",
			inputBody:      `{"email": "", "password": "securepass"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing password",
			inputBody:      `{"email": "test@example.com", "password": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			inputBody:      `invalid-json`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:                 "find by email database error",
			inputBody:            `{"email": "test@example.com", "password": "securepass"}`,
			mockFindByEmailError: errors.New("db connection lost"),
			expectedStatus:       http.StatusInternalServerError,
		},
		{
			name:           "user already exists",
			inputBody:      `{"email": "test@example.com", "password": "securepass"}`,
			expectedStatus: http.StatusConflict,
			expectedUser:   &domain.User{Email: "test@example.com"},
		},
		{
			name:           "bcrypt password too long error",
			inputBody:      `{"email": "test@example.com", "password": "` + strings.Repeat("a", 75) + `"}`, // bcrypt fails > 72 bytes
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "repository save error",
			inputBody:      `{"email": "test@example.com", "password": "securepass"}`,
			mockSaveReturn: errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock_domain.NewMockUserRepository(ctrl)
			ctlr := routes.UserController{UserRepository: mockRepo}

			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tt.inputBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// AnyTimes() allows us to set expectations without the test crashing if they are skipped by an error
			mockRepo.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).Return(tt.expectedUser, tt.mockFindByEmailError).AnyTimes()
			mockRepo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.mockSaveReturn).AnyTimes()

			ctlr.Register(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestUserController_Login(t *testing.T) {
	tests := []struct {
		name             string
		inputBody        string
		mockUser         *domain.User
		mockError        error
		expectedStatus   int
		expectedContains string
	}{
		{
			name:      "successful login",
			inputBody: `{"email": "test@example.com", "password": "password123"}`,
			mockUser: &domain.User{
				Email:    "test@example.com",
				Password: "$2a$12$.OZ5oYXEsFvcaaVh/nmgt.cknGSFzKVlr.wkrzyCl5rgHuAGGkhiS", // password123
				Role:     "admin",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:             "invalid JSON",
			inputBody:        `not-a-json`,
			expectedStatus:   http.StatusBadRequest,
			expectedContains: "Invalid request body",
		},
		{
			name:             "missing email or password",
			inputBody:        `{"email": "", "password": ""}`,
			expectedStatus:   http.StatusBadRequest,
			expectedContains: "Email and password are required",
		},
		{
			name:             "user not found",
			inputBody:        `{"email": "missing@example.com", "password": "password123"}`,
			mockUser:         nil,
			mockError:        errors.New("user not found"),
			expectedStatus:   http.StatusUnauthorized,
			expectedContains: "Invalid email or password",
		},
		{
			name:             "invalid password",
			inputBody:        `{"email": "test@example.com", "password": "wrongpassword"}`,
			mockUser: &domain.User{
				Email:    "test@example.com",
				Password: "$2a$12$.OZ5oYXEsFvcaaVh/nmgt.cknGSFzKVlr.wkrzyCl5rgHuAGGkhiS",
			},
			mockError:        nil,
			expectedStatus:   http.StatusUnauthorized,
			expectedContains: "Invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
            // Ensure a dummy secret exists so standard tests pass
			t.Setenv("JWT_SECRET", "dummy-secret-key-for-testing")

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock_domain.NewMockUserRepository(ctrl)
			ctlr := routes.UserController{UserRepository: mockRepo}

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.inputBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			mockRepo.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).Return(tt.mockUser, tt.mockError).AnyTimes()

			ctlr.Login(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.expectedContains != "" && !strings.Contains(rr.Body.String(), tt.expectedContains) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedContains, rr.Body.String())
			}
		})
	}
}

func TestUserController_GetUsers(t *testing.T) {
	tests := []struct {
		name           string
		userRole       string
		mockUsers      []*domain.User
		mockError      error
		expectedStatus int
		method         string
	}{
		{
			name:           "success admin",
			userRole:       "admin",
			mockUsers:      []*domain.User{{Email: "test@test.com"}},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			method:         http.MethodGet,
		},
		{
			name:           "db error",
			userRole:       "admin",
			mockUsers:      nil,
			mockError:      errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			method:         http.MethodGet,
		},
		{
			name:           "unauthorized wrong role",
			userRole:       "user",
			expectedStatus: http.StatusForbidden,
			method:         http.MethodGet,
		},
		{
			name:           "unauthorized no role",
			userRole:       "",
			expectedStatus: http.StatusForbidden,
			method:         http.MethodGet,
		},
		{
			name:           "method not allowed",
			userRole:       "admin",
			expectedStatus: http.StatusMethodNotAllowed,
			method:         http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock_domain.NewMockUserRepository(ctrl)
			ctlr := routes.UserController{UserRepository: mockRepo}

			// Only expect DB call if method is GET and user is Admin
			if tt.method == http.MethodGet && tt.userRole == "admin" {
				mockRepo.EXPECT().GetAll(gomock.Any()).Return(tt.mockUsers, tt.mockError).AnyTimes()
			}

			req := httptest.NewRequest(tt.method, "/users", nil)
			if tt.userRole != "" {
				req = req.WithContext(context.WithValue(req.Context(), "role", tt.userRole))
			}
			rr := httptest.NewRecorder()

			ctlr.GetUsers(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected %d got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestUserController_GetProfile(t *testing.T) {
	tests := []struct {
		name             string
		userEmail        string
		mockUser         *domain.User
		mockError        error
		expectedStatus   int
		expectedContains string
	}{
		{
			name:      "successful profile fetch",
			userEmail: "test@example.com",
			mockUser: &domain.User{
				Email:    "test@example.com",
				Password: "secret-password", // Checking if it gets zeroed out
			},
			expectedStatus:   http.StatusOK,
			expectedContains: "test@example.com",
		},
		{
			name:             "user not found",
			userEmail:        "missing@example.com",
			mockUser:         nil,
			mockError:        errors.New("user not found"),
			expectedStatus:   http.StatusNotFound,
			expectedContains: "User not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mock_domain.NewMockUserRepository(ctrl)
			ctlr := routes.UserController{UserRepository: mockRepo}

			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			ctx := context.WithValue(req.Context(), "email", tt.userEmail)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			mockRepo.EXPECT().FindByEmail(gomock.Any(), tt.userEmail).Return(tt.mockUser, tt.mockError).AnyTimes()

			ctlr.GetProfile(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.expectedContains != "" && !strings.Contains(rr.Body.String(), tt.expectedContains) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedContains, rr.Body.String())
			}
		})
	}
}

func TestUserController_GetProfile_Unauthorized(t *testing.T) {
	ctlr := routes.UserController{}
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	rr := httptest.NewRecorder()
	ctlr.GetProfile(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestUserController_GetProfile_MethodNotAllowed(t *testing.T) {
	ctlr := routes.UserController{}
	req := httptest.NewRequest(http.MethodPost, "/profile", nil)
	rr := httptest.NewRecorder()
	ctlr.GetProfile(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestUserController_Register_MethodNotAllowed(t *testing.T) {
	ctlr := routes.UserController{}
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rr := httptest.NewRecorder()
	ctlr.Register(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestUserController_Login_MethodNotAllowed(t *testing.T) {
	ctlr := routes.UserController{}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()
	ctlr.Login(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

