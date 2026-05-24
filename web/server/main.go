package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mqtt-streaming-server/broker"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/ocr"
	"mqtt-streaming-server/routes"
)

func NewTLSConfig() *tls.Config {
	// Încărcarea certificatului CA
	certpool := x509.NewCertPool()
	pemCerts, err := os.ReadFile("/run/secrets/ca.crt")
	if err != nil {
		panic(err)
	}
	certpool.AppendCertsFromPEM(pemCerts)

	// Încărcarea certificatului de client
	cert, err := tls.LoadX509KeyPair("/run/secrets/web.crt", "/run/secrets/web.key")
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		RootCAs:            certpool,
		ClientCAs:          certpool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}
}

func main() {
	// Connect to PostgreSQL using GORM
	dsn := fmt.Sprintf("host=postgres-db user=%s password=%s dbname=%s port=5432 sslmode=verify-full sslrootcert=/run/secrets/ca.crt",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	var db *gorm.DB
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to PostgreSQL (attempt %d/%d): %v\n", i+1, maxRetries, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL after multiple attempts:", err)
		panic(err)
	}

	// Auto Migration
	err = db.AutoMigrate(&domain.User{}, &domain.Device{}, &domain.Photo{})
	if err != nil {
		fmt.Println("Failed to run auto-migration:", err)
		panic(err)
	}

	fmt.Println("Connected to PostgreSQL and ran auto-migrations!")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Initialize OCR client connected to the remote OCR service
	ocrHost := os.Getenv("OCR_SERVICE_HOST")
	if ocrHost == "" {
		ocrHost = "ocr-service"
	}
	ocrPort := os.Getenv("OCR_SERVICE_PORT")
	if ocrPort == "" {
		ocrPort = "50051"
	}

	ocrClient, err := ocr.NewClient(ocrHost, ocrPort)
	if err != nil {
		fmt.Printf("Failed to connect to OCR service: %v\n", err)
		panic(err)
	}
	defer ocrClient.Close()

	brokerHandler := broker.NewBrokerHandler(db, ocrClient)

	tlsconfig := NewTLSConfig()

	opts := mqtt.NewClientOptions()
	opts.AddBroker("ssl://broker:8883")
	opts.SetClientID("web").SetTLSConfig(tlsconfig)

	// Start the connection
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to topics
	if token := client.Subscribe("ssproject/images/#", 0, brokerHandler.HandlePhoto); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("register/#", 0, brokerHandler.RegisterDevice); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("auth/login/#", 0, brokerHandler.HandleLogin); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("device/id/#", 0, brokerHandler.DisconnectDevice); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Initialize routes
	handler := routes.InitRoutes(db, client, ocrClient)

	go func() {
		fmt.Println("Starting HTTP server on port 8080...")
		if err := http.ListenAndServe(":8080", handler); err != nil {
			panic(err)
		}
	}()

	go func() {
		fmt.Println("Starting HTTPS server on port 8443...")
		if err := http.ListenAndServeTLS(
			":8443",
			"/run/secrets/server.crt",
			"/run/secrets/server.key",
			handler,
		); err != nil {
			panic(err)
		}
	}()

	<-c
}
