package routes // Changed from routes_test to give us access to the private mqttClient!

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
)

// --- DUMMY MQTT CLIENTS TO PREVENT CRASHES ---
type dummyToken struct{ mqtt.Token }
func (d dummyToken) Wait() bool   { return true }
func (d dummyToken) Error() error { return nil }

type dummyMQTTClient struct{ mqtt.Client }
func (m dummyMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return dummyToken{}
}

// Simulates an MQTT broker failure
type errorToken struct{ mqtt.Token }
func (e errorToken) Wait() bool   { return true }
func (e errorToken) Error() error { return errors.New("mqtt publish failed") }

type errorMQTTClient struct{ mqtt.Client }
func (m errorMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	return errorToken{}
}
// ---------------------------------------------

func TestInitDeviceRoutes(t *testing.T) {
	mux := http.NewServeMux()
	InitDeviceRoutes(nil, dummyMQTTClient{}, mux)
}

func TestDeviceController_GetDevices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockDeviceRepository(ctrl)
	ctlr := DeviceController{DeviceRepository: mockRepo, mqttClient: dummyMQTTClient{}}

	// Success (Admin)
	mockRepo.EXPECT().GetAllDevices(gomock.Any()).Return([]*domain.Device{{UserEmail: "user@example.com"}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	req = req.WithContext(context.WithValue(context.WithValue(req.Context(), "email", "admin@example.com"), "role", "admin"))
	rr := httptest.NewRecorder()
	ctlr.GetDevices(rr, req)

	// Success (User)
	mockRepo.EXPECT().GetAllDevices(gomock.Any()).Return([]*domain.Device{{UserEmail: "user@example.com"}, {UserEmail: "other@example.com"}}, nil)
	req2 := httptest.NewRequest(http.MethodGet, "/devices", nil)
	req2 = req2.WithContext(context.WithValue(context.WithValue(req2.Context(), "email", "user@example.com"), "role", "user"))
	rr2 := httptest.NewRecorder()
	ctlr.GetDevices(rr2, req2)

	// DB Error
	mockRepo.EXPECT().GetAllDevices(gomock.Any()).Return(nil, errors.New("db error"))
	req3 := httptest.NewRequest(http.MethodGet, "/devices", nil)
	rr3 := httptest.NewRecorder()
	ctlr.GetDevices(rr3, req3)

	// Method Not Allowed
	req4 := httptest.NewRequest(http.MethodPost, "/devices", nil)
	rr4 := httptest.NewRecorder()
	ctlr.GetDevices(rr4, req4)
}

func TestDeviceController_SwitchDeviceMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockDeviceRepository(ctrl)

	ctlr := DeviceController{DeviceRepository: mockRepo, mqttClient: dummyMQTTClient{}}
	ctlrErr := DeviceController{DeviceRepository: mockRepo, mqttClient: errorMQTTClient{}}

	// 1. Success
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1", UserEmail: "admin@example.com"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/devices/switch", strings.NewReader(`{"id":"dev-1","mode":"active"}`))
	req = req.WithContext(context.WithValue(context.WithValue(req.Context(), "email", "admin@example.com"), "role", "admin"))
	rr := httptest.NewRecorder()
	ctlr.SwitchDeviceMode(rr, req)

	// 2. MQTT Publish Error
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1", UserEmail: "admin@example.com"}, nil)
	reqErr := httptest.NewRequest(http.MethodPost, "/devices/switch", strings.NewReader(`{"id":"dev-1","mode":"active"}`))
	reqErr = reqErr.WithContext(context.WithValue(context.WithValue(reqErr.Context(), "email", "admin@example.com"), "role", "admin"))
	rrErr := httptest.NewRecorder()
	ctlrErr.SwitchDeviceMode(rrErr, reqErr)

	// 3. Unauthorized
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1", UserEmail: "other@example.com"}, nil)
	reqUnauth := httptest.NewRequest(http.MethodPost, "/devices/switch", strings.NewReader(`{"id":"dev-1","mode":"active"}`))
	reqUnauth = reqUnauth.WithContext(context.WithValue(context.WithValue(reqUnauth.Context(), "email", "user@example.com"), "role", "user"))
	rrUnauth := httptest.NewRecorder()
	ctlr.SwitchDeviceMode(rrUnauth, reqUnauth)

	// 4. Device Not Found
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-missing").Return(nil, errors.New("not found"))
	reqNotFound := httptest.NewRequest(http.MethodPost, "/devices/switch", strings.NewReader(`{"id":"dev-missing","mode":"active"}`))
	rrNotFound := httptest.NewRecorder()
	ctlr.SwitchDeviceMode(rrNotFound, reqNotFound)

	// 5. Invalid JSON
	reqBad := httptest.NewRequest(http.MethodPost, "/devices/switch", strings.NewReader(`invalid`))
	rrBad := httptest.NewRecorder()
	ctlr.SwitchDeviceMode(rrBad, reqBad)

	// 6. Method Not Allowed
	reqGet := httptest.NewRequest(http.MethodGet, "/devices/switch", nil)
	rrGet := httptest.NewRecorder()
	ctlr.SwitchDeviceMode(rrGet, reqGet)
}

func TestDeviceController_SendCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockDeviceRepository(ctrl)
	ctlr := DeviceController{DeviceRepository: mockRepo, mqttClient: dummyMQTTClient{}}
	ctlrErr := DeviceController{DeviceRepository: mockRepo, mqttClient: errorMQTTClient{}}

	// 1. Success
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/devices/command", strings.NewReader(`{"device_id":"dev-1", "command":"CAPTURE"}`))
	req = req.WithContext(context.WithValue(context.WithValue(req.Context(), "email", "admin@example.com"), "role", "admin"))
	rr := httptest.NewRecorder()
	ctlr.SendCommand(rr, req)

	// 2. MQTT Publish Error
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1"}, nil)
	reqErr := httptest.NewRequest(http.MethodPost, "/devices/command", strings.NewReader(`{"device_id":"dev-1", "command":"CAPTURE"}`))
	reqErr = reqErr.WithContext(context.WithValue(context.WithValue(reqErr.Context(), "email", "admin@example.com"), "role", "admin"))
	rrErr := httptest.NewRecorder()
	ctlrErr.SendCommand(rrErr, reqErr)

	// 3. Invalid Command
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1"}, nil)
	reqInv := httptest.NewRequest(http.MethodPost, "/devices/command", strings.NewReader(`{"device_id":"dev-1", "command":"INVALID"}`))
	reqInv = reqInv.WithContext(context.WithValue(context.WithValue(reqInv.Context(), "email", "admin@example.com"), "role", "admin"))
	rrInv := httptest.NewRecorder()
	ctlr.SendCommand(rrInv, reqInv)

	// 4. Unauthorized
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1", UserEmail: "other@example.com"}, nil)
	reqUnauth := httptest.NewRequest(http.MethodPost, "/devices/command", strings.NewReader(`{"device_id":"dev-1", "command":"CAPTURE"}`))
	reqUnauth = reqUnauth.WithContext(context.WithValue(context.WithValue(reqUnauth.Context(), "email", "user@example.com"), "role", "user"))
	rrUnauth := httptest.NewRecorder()
	ctlr.SendCommand(rrUnauth, reqUnauth)

	// 5. Device Not Found
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-missing").Return(nil, errors.New("not found"))
	reqNotFound := httptest.NewRequest(http.MethodPost, "/devices/command", strings.NewReader(`{"device_id":"dev-missing", "command":"CAPTURE"}`))
	rrNotFound := httptest.NewRecorder()
	ctlr.SendCommand(rrNotFound, reqNotFound)

	// 6. Invalid JSON
	reqBad := httptest.NewRequest(http.MethodPost, "/devices/command", strings.NewReader(`invalid`))
	rrBad := httptest.NewRecorder()
	ctlr.SendCommand(rrBad, reqBad)

	// 7. Method Not Allowed
	reqGet := httptest.NewRequest(http.MethodGet, "/devices/command", nil)
	rrGet := httptest.NewRecorder()
	ctlr.SendCommand(rrGet, reqGet)
}

func TestDeviceController_ClaimDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockDeviceRepository(ctrl)
	ctlr := DeviceController{DeviceRepository: mockRepo, mqttClient: dummyMQTTClient{}}

	// 1. Success
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1"}, nil)
    // FIX: Changed "dev-1" to gomock.Any() so it stops crashing on empty strings!
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/devices/claim", strings.NewReader(`{"device_id":"dev-1"}`))
	req = req.WithContext(context.WithValue(req.Context(), "email", "user@example.com"))
	rr := httptest.NewRecorder()
	ctlr.ClaimDevice(rr, req)

	// 2. Conflict (Already claimed)
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1", UserEmail: "other@example.com"}, nil)
	reqConf := httptest.NewRequest(http.MethodPost, "/devices/claim", strings.NewReader(`{"device_id":"dev-1"}`))
	reqConf = reqConf.WithContext(context.WithValue(reqConf.Context(), "email", "user@example.com"))
	rrConf := httptest.NewRecorder()
	ctlr.ClaimDevice(rrConf, reqConf)

	// 3. Update DB Error
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-1").Return(&domain.Device{ID: "dev-1"}, nil)
    // FIX: Changed "dev-1" to gomock.Any() here too!
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))
	reqDbErr := httptest.NewRequest(http.MethodPost, "/devices/claim", strings.NewReader(`{"device_id":"dev-1"}`))
	reqDbErr = reqDbErr.WithContext(context.WithValue(reqDbErr.Context(), "email", "user@example.com"))
	rrDbErr := httptest.NewRecorder()
	ctlr.ClaimDevice(rrDbErr, reqDbErr)

	// 4. Device Not Found
	mockRepo.EXPECT().GetByID(gomock.Any(), "dev-missing").Return(nil, errors.New("not found"))
	reqNotFound := httptest.NewRequest(http.MethodPost, "/devices/claim", strings.NewReader(`{"device_id":"dev-missing"}`))
	rrNotFound := httptest.NewRecorder()
	ctlr.ClaimDevice(rrNotFound, reqNotFound)

	// 5. Invalid JSON
	reqBad := httptest.NewRequest(http.MethodPost, "/devices/claim", strings.NewReader(`invalid`))
	rrBad := httptest.NewRecorder()
	ctlr.ClaimDevice(rrBad, reqBad)

	// 6. Method Not Allowed
	reqGet := httptest.NewRequest(http.MethodGet, "/devices/claim", nil)
	rrGet := httptest.NewRecorder()
	ctlr.ClaimDevice(rrGet, reqGet)
}

