package broker_test

import (
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gorm.io/gorm"

	"mqtt-streaming-server/broker"
)

func TestBrokerHandler_RegisterDevice(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		db        *gorm.DB
		ocrClient broker.OCRClientInterface
		// Named input parameters for target function.
		msg mqtt.Message
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := broker.NewBrokerHandler(tt.db, tt.ocrClient)
			b.RegisterDevice(nil, tt.msg)
		})
	}
}
