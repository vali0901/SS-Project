package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/otiai10/gosseract/v2"
	"gorm.io/gorm"

	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/repository"
	"mqtt-streaming-server/utils"
)

type BrokerHandler struct {
	photoRepository  domain.PhotoRepository
	deviceRepository domain.DeviceRepository
	userRepository   domain.UserRepository
	ocrClient        *gosseract.Client
}

func NewBrokerHandler(db *gorm.DB, ocrClient *gosseract.Client) BrokerHandler {
	return BrokerHandler{
		photoRepository:  repository.NewPhotoRepository(db),
		deviceRepository: repository.NewDeviceRepository(db),
		userRepository:   repository.NewUserRepository(db),
		ocrClient:        ocrClient,
	}
}

func (b BrokerHandler) HandleLogin(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	// topic is auth/login/device_id
	if len(topic) <= len("auth/login/") {
		return
	}
	deviceID := topic[len("auth/login/"):]
	ctx := context.Background()

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		fmt.Printf("Invalid login request from %s: %v\n", deviceID, err)
		return
	}

	// Authenticate user
	user, err := b.userRepository.FindByEmail(ctx, req.Email)
	if err != nil {
		fmt.Printf("Login failed for %s: user not found\n", req.Email)
		return
	}

	// Use bcrypt to check password
	// We need "golang.org/x/crypto/bcrypt" imported
	if err := utils.CheckPassword(user.Password, req.Password); err != nil {
		fmt.Printf("Login failed for %s: invalid password\n", req.Email)
		return
	}

	// Generate token
	token, err := utils.GenerateToken(user.Email, user.Role)
	if err != nil {
		fmt.Printf("Failed to generate token for %s: %v\n", req.Email, err)
		return
	}

	// Respond with token
	responseTopic := fmt.Sprintf("auth/login/response/%s", deviceID)
	response := map[string]string{
		"token": token,
		"email": user.Email,
	}
	payload, _ := json.Marshal(response)
	client.Publish(responseTopic, 0, false, payload)
	fmt.Printf("User %s logged in over MQTT for device %s\n", req.Email, deviceID)
}

func (b BrokerHandler) HandlePhoto(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	var deviceID string
	// topic is ssproject/images/device_id or just ssproject/images
	if topic == "ssproject/images" {
		deviceID = "camera_stream"
	} else if len(topic) > len("ssproject/images/") {
		deviceID = topic[len("ssproject/images/"):]
	} else {
		deviceID = "unknown"
	}

	ctx := context.Background()
	fmt.Println("Received message on topic:", msg.Topic())

	// get registered device
	device, err := b.deviceRepository.GetByID(ctx, deviceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("Unauthorized photo upload: Device ID %s not registered. Rejecting.\n", deviceID)
			return
		}
		fmt.Printf("Failed to check device ID: %v\n", err)
		return
	}

	if device.UserEmail == "" {
		fmt.Printf("Unauthorized photo upload: Device %s is not claimed by any user. Rejecting.\n", deviceID)
		return
	}

	fmt.Printf("Received photo from device: %s (User: %s)\n", device.DeviceName, device.UserEmail)
	body := msg.Payload()
	_, imageType, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Failed to decode image: %v\n", err)
		return
	}
	fmt.Printf("Image type: %s\n", imageType)

	// Extract text from image
	text, err := b.extractTextFromImage(body)
	if err != nil {
		fmt.Printf("Failed to extract text from image: %v\n", err)
		text = "OCR failed"
	}

	// Try to extract structured medical data
	var medicalData *domain.MedicalData
	if utils.IsMedicalCertificate(text) {
		medicalData = utils.ParseMedicalCertificate(text)
		if medicalData != nil {
			fmt.Printf("Extracted medical data: %+v\n", medicalData)
		}
	}

	// UTC timestamp
	timestamp := time.Now().UTC()

	// Create photo with embedded medical data
	photo := &domain.Photo{
		ID:        uuid.New().String(),
		ImageType: imageType,
		Timestamp: timestamp,
		DeviceID:  deviceID,
		UserEmail: device.UserEmail,
		Text:      text,
	}

	// Copy medical data fields directly to photo
	if medicalData != nil {
		photo.MedicalData = *medicalData
	}

	err = b.photoRepository.Save(ctx, photo)
	if err != nil {
		fmt.Printf("Failed to insert photo into PostgreSQL: %v\n", err)
		return
	}
	// Save photo locally
	keyName := utils.PhotoStorageKey(photo.ID)
	if err := utils.SaveToLocal(body, keyName); err != nil {
		fmt.Printf("Failed to save photo locally: %v\n", err)
		return
	}
	fmt.Printf("Photo saved locally with key: %s\n", keyName)
}

func (b BrokerHandler) RegisterDevice(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	// topic is register/device_id
	deviceID := topic[len("register/"):]
	ctx := context.Background()
	fmt.Println("Received message on topic:", msg.Topic())
	body := msg.Payload()
	fmt.Printf("Received device registration: %s\n", body)

	// Parse JSON payload: {"name": "...", "ip": "...", "port": "...", "token": "..."}
	var deviceName, ipAddress, port, token string
	var registration struct {
		Name  string `json:"name"`
		IP    string `json:"ip"`
		Port  string `json:"port"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &registration); err == nil {
		deviceName = registration.Name
		ipAddress = registration.IP
		port = registration.Port
		token = registration.Token
	}

	if deviceName == "" {
		deviceName = string(body)
	}

	// Authenticate with JWT token if provided
	var userEmail string
	if token != "" {
		claims, err := utils.VerifyToken(token)
		if err == nil {
			userEmail = claims.Email
			fmt.Printf("Authenticated user from token: %s\n", userEmail)
		} else {
			fmt.Printf("Invalid token provided in registration: %v\n", err)
			// For now we might still allow registration but without owner,
			// or we could reject it. User said "requires authentification".
			return
		}
	} else {
		fmt.Println("No token provided in registration. Rejecting.")
		return
	}

	// Check if device ID already exists
	existingDevice, err := b.deviceRepository.GetByID(ctx, deviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		fmt.Printf("Failed to check device ID: %v\n", err)
		return
	}

	if err == gorm.ErrRecordNotFound {
		// Device ID does not exist, insert it
		err = b.deviceRepository.Save(ctx, &domain.Device{
			ID:           deviceID,
			DeviceID:     deviceID,
			DeviceName:   deviceName,
			DeviceStatus: "active",
			IPAddress:    ipAddress,
			Port:         port,
			UserEmail:    userEmail,
			LastSeen:     time.Now().UTC(),
		})
		if err != nil {
			fmt.Printf("Failed to insert device ID: %v\n", err)
			return
		}
		fmt.Printf("Device registered and claimed by %s: %s (IP: %s, Port: %s)\n", userEmail, deviceID, ipAddress, port)
		return
	}

	// Device ID already exists
	// Only allow update if same user or device is unclaimed
	if existingDevice.UserEmail != "" && existingDevice.UserEmail != userEmail {
		fmt.Printf("Unauthorized attempt to update device %s by user %s\n", deviceID, userEmail)
		return
	}

	err = b.deviceRepository.Update(ctx, deviceID, &domain.Device{
		DeviceName:   deviceName,
		DeviceStatus: "active",
		IPAddress:    ipAddress,
		Port:         port,
		UserEmail:    userEmail,
		LastSeen:     time.Now().UTC(),
	})
	if err != nil {
		fmt.Printf("Failed to update device ID: %v\n", err)
		return
	}
	fmt.Printf("Device updated and claimed by %s: %s (IP: %s, Port: %s)\n", userEmail, deviceID, ipAddress, port)
}

func (b BrokerHandler) DisconnectDevice(_ mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	// topic is device/id/device_id
	var deviceID string
	if len(topic) > len("device/id/") {
		deviceID = topic[len("device/id/"):]
	} else {
		return
	}

	ctx := context.Background()
	fmt.Println("Received message on topic:", msg.Topic())
	message := string(msg.Payload())
	fmt.Printf("Received device disconnection: %s\n", message)

	if message != "Device Disconnected" {
		fmt.Printf("Invalid disconnection message: %s\n", message)
		return
	}

	device, err := b.deviceRepository.GetByID(ctx, deviceID)
	if err != nil {
		// handle error
		return
	}
	if device.DeviceStatus != "active" {
		return
	}
	err = b.deviceRepository.Update(ctx, deviceID, &domain.Device{
		DeviceStatus: "inactive",
	})
	if err != nil {
		fmt.Printf("Failed to mark device as inactive: %v\n", err)
	}
}

func (b BrokerHandler) extractTextFromImage(imageData []byte) (string, error) {
	// Use the OCR client to extract text from the image
	b.ocrClient.SetImageFromBytes(imageData)
	text, err := b.ocrClient.Text()
	if err != nil {
		return "", fmt.Errorf("failed to extract text from image: %v", err)
	}
	return text, nil
}

func (b BrokerHandler) HandleCommand(_ mqtt.Client, msg mqtt.Message) {
	fmt.Println("Received command on topic:", msg.Topic())
	body := string(msg.Payload())
	fmt.Printf("Command payload: %s\n", body)
}
