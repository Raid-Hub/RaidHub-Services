package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"raidhub/shared/monitoring"
	"raidhub/shared/postgres"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Set up RabbitMQ connection
	username := os.Getenv("RABBITMQ_USER")
	password := os.Getenv("RABBITMQ_PASSWORD")
	port := os.Getenv("RABBITMQ_PORT")

	if username == "" || password == "" || port == "" {
		log.Fatalf("Environment variables RABBITMQ_USER, RABBITMQ_PASSWORD, or RABBITMQ_PORT are not set")
	}

	rabbitURL := fmt.Sprintf("amqp://%s:%s@localhost:%s/", username, password, port)
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	// Declare the queue
	queueName := "player_requests"
	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %s", err)
	}

	// Set up PostgreSQL connection
	db, err := postgres.Connect()
	if err != nil {
		log.Fatalf("Error connecting to the database: %s", err)
	}
	defer db.Close()

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %s", err)
	}

	// Handle incoming messages
	go func() {
		for msg := range msgs {
			var request PlayerRequest
			if err := json.Unmarshal(msg.Body, &request); err != nil {
				log.Printf("Failed to unmarshal message: %s", err)
				continue
			}
			processRequest(&request, db)
		}
	}()

	monitoring.RegisterPrometheus(8083)

	// Keep the main thread running
	log.Println("Waiting for messages...")
	forever := make(chan bool)
	<-forever
}
