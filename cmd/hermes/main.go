package main

import (
	"log"
	"raidhub/async/activity_history"
	"raidhub/async/bonus_pgcr"
	"raidhub/async/character_fill"
	"raidhub/async/player_crawl"
	"raidhub/shared/clickhouse"
	"raidhub/shared/monitoring"
	"raidhub/shared/pgcr"
	"raidhub/shared/postgres"
	"raidhub/shared/rabbit"
)

func main() {
	db, err := postgres.Connect()
	if err != nil {
		log.Fatal("Error connecting to postgres", err)
	}
	defer db.Close()

	conn, err := rabbit.Init()
	if err != nil {
		log.Fatal("Error connecting to rabbit", err)
	}
	defer rabbit.Cleanup()

	chClient, err := clickhouse.Connect(false)
	if err != nil {
		log.Fatal("Error connecting to clickhouse", err)
	}
	defer chClient.Close()

	activityHistoryQueue := activity_history.Create()
	activityHistoryQueue.Conn = conn
	activityHistoryQueue.Db = db
	go activityHistoryQueue.Register(3)

	playersQueue := player_crawl.Create()
	playersQueue.Db = db
	playersQueue.Conn = conn
	go playersQueue.Register(10)

	activityCharactersQueue := character_fill.Create()
	activityCharactersQueue.Db = db
	activityCharactersQueue.Conn = conn
	go activityCharactersQueue.Register(5)

	pgcrsClickhouseQueue := pgcr.CreateClickhouseQueue()
	pgcrsClickhouseQueue.Db = db
	pgcrsClickhouseQueue.Conn = conn
	pgcrsClickhouseQueue.Clickhouse = &chClient
	go pgcrsClickhouseQueue.Register(1)

	bonusPgcrsFetchQueue := bonus_pgcr.CreateFetchWorker()
	bonusPgcrsFetchQueue.Db = db
	bonusPgcrsFetchQueue.Conn = conn
	go bonusPgcrsFetchQueue.Register(25)

	bonusPgcrsStoreQueue := bonus_pgcr.CreateStoreWorker()
	bonusPgcrsStoreQueue.Db = db
	bonusPgcrsStoreQueue.Conn = conn
	// 1 worker because it's a write operation with often related records which would cause deadlocks
	go bonusPgcrsStoreQueue.Register(1)

	monitoring.RegisterPrometheus(8083)

	// Keep the main thread running
	forever := make(chan bool)
	<-forever
}
