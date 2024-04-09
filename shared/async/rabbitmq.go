package async

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"raidhub/shared/postgres"
	"sync"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	conn *amqp.Connection
	once sync.Once
)

func Init() (*amqp.Connection, error) {
	var err error = nil
	once.Do(func() {
		conn, err = initOnce()
	})
	return conn, err
}

func initOnce() (*amqp.Connection, error) {
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

	return amqp.Dial(rabbitURL)
}

func Cleanup() {
	// Clean up the resources when the program exits
	if conn != nil {
		conn.Close()
	}
}

func RegisterQueueWorker(queueName string, numWorkers int, worker func(msgs <-chan amqp.Delivery, db *sql.DB)) {
	conn, err := Init()
	if err != nil {
		log.Fatalf("Failed to establish connection: %s", err)
	}
	defer Cleanup()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %s", err)
	}
	defer ch.Close()

	// Set up PostgreSQL connection
	db, err := postgres.Connect()
	if err != nil {
		log.Fatalf("Error connecting to the database: %s", err)
	}
	defer db.Close()

	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create queue: %s", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %s", err)
	}

	for i := 0; i < numWorkers; i++ {
		go worker(msgs, db)
	}

	log.Printf("Waiting for messages on queue %s...", queueName)
	forever := make(chan bool)
	<-forever
}

// // Open a channel
// ch, err = conn.Channel()
// if err != nil {
// 	conn.Close()
// 	return nil, nil, err
// }

// return conn, ch, nil
