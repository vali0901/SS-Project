package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gorm.io/gorm"

	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/repository"
)

type DeviceController struct {
	DeviceRepository domain.DeviceRepository
	mqttClient       mqtt.Client
}

func InitDeviceRoutes(db *gorm.DB, mqttClient mqtt.Client, mux *http.ServeMux) {
	deviceController := &DeviceController{
		DeviceRepository: repository.NewDeviceRepository(db),
		mqttClient:       mqttClient,
	}

	mux.Handle("/devices", withAuth(http.HandlerFunc(deviceController.GetDevices)))
	mux.Handle("/devices/claim", withAuth(http.HandlerFunc(deviceController.ClaimDevice)))
	mux.Handle("/devices/switch", withAuth(http.HandlerFunc(deviceController.SwitchDeviceMode)))
	mux.Handle("/devices/command", withAuth(http.HandlerFunc(deviceController.SendCommand)))
}

func (ctlr DeviceController) ClaimDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)

	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if device exists
	device, err := ctlr.DeviceRepository.GetByID(ctx, req.DeviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Check if already claimed
	if device.UserEmail != "" && device.UserEmail != userEmail {
		http.Error(w, "Device already claimed by another user", http.StatusConflict)
		return
	}

	// Update ownership
	device.UserEmail = userEmail
	if err := ctlr.DeviceRepository.Update(ctx, device.DeviceID, device); err != nil {
		http.Error(w, "Failed to claim device", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(device)
}

func (ctlr DeviceController) SwitchDeviceMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)
	role, _ := ctx.Value("role").(string)

	var req struct {
		ID   string `json:"id"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check ownership
	device, err := ctlr.DeviceRepository.GetByID(ctx, req.ID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	if role != "admin" && device.UserEmail != userEmail {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	topic := fmt.Sprintf("setup/%s", device.ID)
	if token := ctlr.mqttClient.Publish(topic, 0, false, "start "+req.Mode); token.Wait() && token.Error() != nil {
		http.Error(w, "Failed to publish message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ctlr DeviceController) GetDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)
	role, _ := ctx.Value("role").(string)

	// Fetch all devices
	devices, err := ctlr.DeviceRepository.GetAllDevices(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}

	// Filter based on ownership if not admin
	filteredDevices := make([]*domain.Device, 0)
	for _, d := range devices {
		if role == "admin" || d.UserEmail == userEmail || d.UserEmail == "" {
			filteredDevices = append(filteredDevices, d)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredDevices)
}

func (ctlr DeviceController) SendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)
	role, _ := ctx.Value("role").(string)

	var request struct {
		DeviceID string `json:"device_id"`
		Command  string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check ownership
	device, err := ctlr.DeviceRepository.GetByID(ctx, request.DeviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	if role != "admin" && device.UserEmail != userEmail {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Validate command
	validCommands := map[string]bool{
		"CAPTURE":    true,
		"START-LIVE": true,
		"STOP-LIVE":  true,
	}
	if !validCommands[request.Command] {
		http.Error(w, "Invalid command. Must be CAPTURE, START-LIVE, or STOP-LIVE", http.StatusBadRequest)
		return
	}

	// Publish command to MQTT topic ssproject/commands
	topic := "ssproject/commands"
	payload := request.Command
	if token := ctlr.mqttClient.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
		http.Error(w, "Failed to publish command", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Command %s sent to device %s", request.Command, request.DeviceID),
	})
}
