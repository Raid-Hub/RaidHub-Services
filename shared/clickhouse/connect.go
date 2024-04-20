package clickhouse

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/ClickHouse/clickhouse-go"
	"github.com/joho/godotenv"
)

type ClickhouseClient struct {
	conn *sql.DB
}

func Connect(debug bool) (*ClickhouseClient, error) {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	port := os.Getenv("CLICKHOUSE_PORT")
	if port == "" {
		port = "9000"
	}

	user := os.Getenv("CLICKHOUSE_USER")
	if user == "" {
		user = "default"
	}

	password := os.Getenv("CLICKHOUSE_PASSWORD")

	clickhouseURI := fmt.Sprintf("tcp://%s:%s?debug=%t&username=%s&password=%s", "localhost", port, debug, user, password)

	connect, err := sql.Open("clickhouse", clickhouseURI)
	if err != nil {
		return nil, err
	}

	if err := connect.Ping(); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			return nil, fmt.Errorf("[%d] %s \n%s", exception.Code, exception.Message, exception.StackTrace)
		} else {
			return nil, err
		}
	}

	return &ClickhouseClient{
		conn: connect,
	}, nil
}

func (c *ClickhouseClient) Close() error {
	return c.conn.Close()
}

func (c *ClickhouseClient) Begin() (*sql.Tx, error) {
	return c.conn.Begin()
}
