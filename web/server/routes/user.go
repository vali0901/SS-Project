package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/repository"
	"mqtt-streaming-server/utils"
)

type UserController struct {
	UserRepository domain.UserRepository
}

func InitUserRoutes(db *gorm.DB, mux *http.ServeMux) {
	userController := &UserController{
		UserRepository: repository.NewUserRepository(db),
	}

	mux.HandleFunc("/register", userController.Register)
	mux.HandleFunc("/login", userController.Login)
	mux.Handle("/users", withAuth(http.HandlerFunc(userController.GetUsers)))
	mux.Handle("/profile", withAuth(http.HandlerFunc(userController.GetProfile)))
}

func (ctlr UserController) GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is admin
	role, ok := r.Context().Value("role").(string)
	if !ok || role != "admin" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Fetch all users
	users, err := ctlr.UserRepository.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	// Clean up passwords before sending
	for _, u := range users {
		u.Password = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (ctlr UserController) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// look for existing user
	existingUser, err := ctlr.UserRepository.FindByEmail(r.Context(), req.Email)
	if err != nil && err != gorm.ErrRecordNotFound {
		http.Error(w, "Failed to check existing user", http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Save the user to the database
	role := req.Role
	if role == "" {
		role = "user"
	}
	err = ctlr.UserRepository.Save(r.Context(), req.Email, string(hashedPassword), role)
	if err != nil {
		fmt.Printf("Error saving user %s: %v\n", req.Email, err)
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	fmt.Printf("User registered successfully: %s (role: %s)\n", req.Email, role)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "User registered successfully")
}

func (ctlr UserController) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if the user exists
	user, err := ctlr.UserRepository.FindByEmail(r.Context(), req.Email)
	if err != nil {
		fmt.Printf("Login failed: user %s not found\n", req.Email)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		fmt.Printf("Login failed: invalid password for user %s\n", req.Email)
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT token using shared utility
	tokenString, err := utils.GenerateToken(user.Email, user.Role)
	if err != nil {
		fmt.Printf("Error generating token for %s: %v\n", user.Email, err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	fmt.Printf("User logged in successfully: %s (role: %s)\n", user.Email, user.Role)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
		"email": user.Email,
		"role":  user.Role,
	})
}

func (ctlr UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok {
		http.Error(w, "Email not found in context", http.StatusUnauthorized)
		return
	}

	// Retrieve the user's profile from the database
	user, err := ctlr.UserRepository.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	// Exclude the password from the response
	user.Password = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
