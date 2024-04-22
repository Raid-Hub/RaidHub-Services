package async

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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

type QueueWorker struct {
	QueueName  string
	Conn       *amqp.Connection
	Db         *sql.DB
	Clickhouse *driver.Conn
	Processer  func(qw *QueueWorker, msgs <-chan amqp.Delivery)
}

func (qw *QueueWorker) Register(numWorkers int) {
	ch, err := qw.Conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %s", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		qw.QueueName,
		true,
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
		go qw.Processer(qw, msgs)
	}

	log.Printf("Waiting for messages on queue %s...", qw.QueueName)
	forever := make(chan bool)
	<-forever
}
