package async

import (
	"database/sql"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	amqp "github.com/rabbitmq/amqp091-go"
)

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

	for i := 0; i < numWorkers; i++ {
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
		go qw.Processer(qw, msgs)
	}

	log.Printf("Waiting for messages on queue %s...", qw.QueueName)
	forever := make(chan bool)
	<-forever
}
