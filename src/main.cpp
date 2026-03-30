/**********************************************************************
  Filename    : Camera MQTT Client
  Description : ESP32-CAM MQTT Image Transfer
**********************************************************************/
#include "esp_camera.h"
#include <WiFiClientSecure.h>
#include <WiFi.h>
#include <PubSubClient.h>
// CAMERA_MODEL is defined in platformio.ini
#include "../include/camera_pins.h"

// ===========================
// Configuration
// ===========================
const char* ssid     = "";       // TODO: Modificați cu SSID-ul rețelei voastre
const char* password = "";     // TODO: Modificați cu parola rețelei voastre
const char* mqtt_server = ""; // TODO: Modificați cu IP-ul calculatorului (ip addr / ipconfig)
const int mqtt_port = 8883;

const char* ca_cert = "\
-----BEGIN CERTIFICATE-----\
MIIDETCCAfmgAwIBAgIUBq4bNA3i3eo6dTNVpY0a8uUrXWowDQYJKoZIhvcNAQEL\
BQAwGDEWMBQGA1UEAwwNU1MtUHJvamVjdC1DQTAeFw0yNjAzMzAxODAxMjVaFw0z\
NjAzMjcxODAxMjVaMBgxFjAUBgNVBAMMDVNTLVByb2plY3QtQ0EwggEiMA0GCSqG\
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDO175w/89lIKy3jcw4jgj4nVQdESLJREFV\
DukCVL2Z6nvZ0xsefNt1GpgMyIoGAmjZ6Wo65ucxKiMAXXNSRH+O0A0p8akXEILq\
NEsn9MsagXPq86jx6v6sEqsws8H39DYIkQx1VSF3gfcRyEbuyhUOLLJqRZuVjplG\
BzyzvJ5J+NJkAcTHCYdepyi6UVv7+h2LeEY1HLuVwivTVEpP+fhhPAIWQOEU2Mvc\
WlLNmh/VCJBfRuaon8wpS9nDqHIr6m6EPJpcjLCDw+kI42v2gaHz+gjMW6IFw4Cc\
e618M2SNMtmn/7KxAlAuf+J7gIYZiu+O34cPVwPRv65ecSy6PoKVAgMBAAGjUzBR\
MB0GA1UdDgQWBBSmCaPboZ1d2spFgGs8DGEDxix+ajAfBgNVHSMEGDAWgBSmCaPb\
oZ1d2spFgGs8DGEDxix+ajAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUA\
A4IBAQA8miRdQyoEX3UJv/hN1BOqy/1iPejd4T4Ea7Ok4lcBXGrBQ4zrGER26bHP\
9jglZa5eDtl6Gxe+IMSiMB2DzEJd4VQNUOuIr/eFduvvdnhfJbLUjo8Bu3dTOI1Q\
kYnAvYwCpkZLj2qW1bhrAbd0N1Lt6TBPLfxQTVzm20wedsumxG4cB0OImUbM2rp5\
Eid+4bmN6tbhbi0Jtklw4gZOaDEuRHDV+aVM5yPquzfk66klcScAQuBlOALLy/2C\
Iqocg06gbEiIks95htbY+HlPLS/HHDh/F7ND55ohvT0m0IOKgwwX20GvNqfo8TjR\
NH56dKsKD514QZtygJRT4Y2mCFM+\
-----END CERTIFICATE-----\
";

// Topics
const char* TOPIC_COMMAND = "ssproject/commands";
const char* TOPIC_IMAGE   = "ssproject/images";

WiFiClientSecure espClient;
PubSubClient client(espClient);

// State variables
bool flash_on = false;
bool streaming = false;
bool take_one_picture = false;
unsigned long last_capture_time = 0;
const unsigned long STREAM_INTERVAL = 100; // ms

void setup_camera() {
  camera_config_t config = {};
  config.ledc_channel    = LEDC_CHANNEL_0;
  config.ledc_timer      = LEDC_TIMER_0;
  config.pin_d0          = Y2_GPIO_NUM;
  config.pin_d1          = Y3_GPIO_NUM;
  config.pin_d2          = Y4_GPIO_NUM;
  config.pin_d3          = Y5_GPIO_NUM;
  config.pin_d4          = Y6_GPIO_NUM;
  config.pin_d5          = Y7_GPIO_NUM;
  config.pin_d6          = Y8_GPIO_NUM;
  config.pin_d7          = Y9_GPIO_NUM;
  config.pin_xclk        = XCLK_GPIO_NUM;
  config.pin_pclk        = PCLK_GPIO_NUM;
  config.pin_vsync       = VSYNC_GPIO_NUM;
  config.pin_href        = HREF_GPIO_NUM;
  config.pin_sccb_sda    = SIOD_GPIO_NUM;
  config.pin_sccb_scl    = SIOC_GPIO_NUM;
  config.pin_pwdn        = PWDN_GPIO_NUM;
  config.pin_reset       = RESET_GPIO_NUM;
  config.xclk_freq_hz    = 20000000;
  config.pixel_format    = PIXFORMAT_JPEG;

  if (psramFound()) {
    Serial.println("PSRAM found!");
    config.frame_size    = FRAMESIZE_VGA;
    config.jpeg_quality  = 12;
    config.fb_count      = 2;
  } else {
    Serial.println("No PSRAM found, using DRAM");
    config.frame_size    = FRAMESIZE_SVGA;
    config.jpeg_quality  = 12;
    config.fb_count      = 1;
    config.fb_location   = CAMERA_FB_IN_DRAM;
  }

  Serial.println("Initializing camera...");
  esp_err_t err = esp_camera_init(&config);
  if (err != ESP_OK) {
    Serial.printf("Camera init failed with error 0x%x\n", err);
    return;
  }
  Serial.println("Camera Ready!");

  pinMode(4, OUTPUT);
}

void callback(char* topic, byte* payload, unsigned int length) {
  Serial.println(">>> CALLBACK FIRED <<<");
  String message;
  for (int i = 0; i < length; i++) {
    message += (char)payload[i];
  }
  Serial.printf("Topic: %s\n", topic);
  Serial.printf("Message: [%s] (len=%u)\n", message.c_str(), length);

  if (String(topic) == TOPIC_COMMAND) {
    if (message == "CAPTURE") {
      take_one_picture = true;
      Serial.println("=> Action: take_one_picture = true");
    } else if (message == "START-LIVE") {
      streaming = true;
      Serial.println("=> Action: Streaming Started");
    } else if (message == "STOP-LIVE") {
      streaming = false;
      Serial.println("=> Action: Streaming Stopped");
    } else if (message == "FLASH") {
      flash_on = !flash_on;
      digitalWrite(4, flash_on ? HIGH : LOW); // GPIO4 controls the flash
      Serial.println("=> Action: FLASH");
    } else {
      Serial.println("=> Unknown command, ignoring");
    }
  } else {
    Serial.println("=> Wrong topic, ignoring");
  }
}

void reconnect() {
  while (!client.connected()) {
    Serial.print("Attempting MQTT connection...");
    String clientId = "ESP32CamClient-";
    clientId += String(random(0xffff), HEX);
    Serial.printf(" (clientId=%s)\n", clientId.c_str());

    if (client.connect(clientId.c_str())) {
      Serial.println("MQTT connected!");
      bool subOk = client.subscribe(TOPIC_COMMAND);
      Serial.printf("Subscribe to '%s': %s\n", TOPIC_COMMAND, subOk ? "OK" : "FAILED");
      Serial.printf("Buffer size: %d\n", client.getBufferSize());
      Serial.printf("Free heap: %u bytes\n", ESP.getFreeHeap());
    } else {
      Serial.print("failed, rc=");
      Serial.print(client.state());
      Serial.println(" try again in 5 seconds");
      delay(5000);
    }
  }
}

void setup() {
  Serial.begin(115200);
  delay(1000);
  Serial.println();
  Serial.println("============================");
  Serial.println("  ESP32-CAM MQTT Client");
  Serial.println("============================");
  Serial.printf("Free heap at start: %u bytes\n", ESP.getFreeHeap());
  Serial.printf("PSRAM size: %u bytes\n", ESP.getPsramSize());

  setup_camera();

  Serial.printf("Connecting to WiFi: %s\n", ssid);
  WiFi.begin(ssid, password);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.printf("\nWiFi connected! IP: %s\n", WiFi.localIP().toString().c_str());

  espClient.setCACert(ca_cert);
  client.setServer(mqtt_server, mqtt_port);
  client.setCallback(callback);
  client.setBufferSize(65000);
}

void captureAndPublish() {
    camera_fb_t * fb = esp_camera_fb_get();
    if (!fb) {
        Serial.println("Camera capture failed");
        return;
    }

    if (client.publish(TOPIC_IMAGE, (const uint8_t*)fb->buf, fb->len)) {
        Serial.printf("Image published: %u bytes\n", fb->len);
    } else {
        Serial.println("Publish failed");
    }

    esp_camera_fb_return(fb);
}

unsigned long last_heartbeat = 0;

void loop() {
  if (!client.connected()) {
    Serial.println("MQTT disconnected, reconnecting...");
    reconnect();
  }
  client.loop();

  unsigned long now = millis();

  // Print a heartbeat every 5 seconds so you know the loop is running
  if (now - last_heartbeat > 5000) {
    Serial.printf("[heartbeat] millis=%lu connected=%d streaming=%d free_heap=%u\n",
                  now, client.connected(), streaming, ESP.getFreeHeap());
    last_heartbeat = now;
  }

  if (take_one_picture) {
    Serial.println("Taking single picture...");
    captureAndPublish();
    take_one_picture = false;
  }

  if (streaming && (now - last_capture_time > STREAM_INTERVAL)) {
    captureAndPublish();
    last_capture_time = now;
  }
}