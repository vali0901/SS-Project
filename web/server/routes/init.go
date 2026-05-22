package routes

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/otiai10/gosseract/v2"
	"gorm.io/gorm"

	"mqtt-streaming-server/utils"
)

func InitRoutes(db *gorm.DB, mqttClient mqtt.Client, ocrClient *gosseract.Client) http.Handler {
	mux := http.NewServeMux()
	InitUserRoutes(db, mux)
	InitPhotoRoutes(db, ocrClient, mux)
	InitDeviceRoutes(db, mqttClient, mux)

	// Serve static files from ./uploads
	// Ensure the directory exists or handle errors gracefully, but FileServer is robust enough.
	fs := http.FileServer(http.Dir("uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// Broker info endpoint - returns the MQTT broker connection info
	mux.HandleFunc("/broker-info", handleBrokerInfo)

	corsHandler := withCORS(mux)

	// Add other middleware here if needed
	return corsHandler
}

// handleBrokerInfo returns the MQTT broker IP and port for client connections
func handleBrokerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the server's local IP address
	ip := getOutboundIP()
	port := "8883" // Default MQTT port mTls

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"ip":   ip,
		"port": port,
	})
}

// getOutboundIP gets the preferred outbound IP of this machine
// In Docker, we need to use the host's external IP, not the container IP
func getOutboundIP() string {
	// First, check if MQTT_HOST_IP is set explicitly
	if hostIP := os.Getenv("MQTT_HOST_IP"); hostIP != "" {
		// If it's a hostname (like host.docker.internal), resolve it
		if addrs, err := net.LookupHost(hostIP); err == nil && len(addrs) > 0 {
			return addrs[0]
		}
		// If it's already an IP, return as-is
		if net.ParseIP(hostIP) != nil {
			return hostIP
		}
	}

	// Try to resolve host.docker.internal (works in Docker Desktop)
	addrs, err := net.LookupHost("host.docker.internal")
	if err == nil && len(addrs) > 0 {
		return addrs[0]
	}

	// Fallback: detect outbound IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace * with your domain in production
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// withAuth is a middleware that validates the JWT token from the Authorization header.
func withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the JWT token from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		if len(authHeader) < len("Bearer ") {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[len("Bearer "):] // Remove "Bearer " prefix
		claims, err := utils.VerifyToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Store the email and role in the request context
		ctx := context.WithValue(r.Context(), "email", claims.Email)
		ctx = context.WithValue(ctx, "role", claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
