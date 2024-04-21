package main

import (
	"log"
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"
	"raidhub/shared/async/bonus_pgcr"
	"raidhub/shared/async/character_fill"
	"raidhub/shared/async/player_crawl"
	"raidhub/shared/clickhouse"
	"raidhub/shared/monitoring"
	"raidhub/shared/pgcr"
	"raidhub/shared/postgres"
)

func main() {
	db, err := postgres.Connect()
	if err != nil {
		log.Fatal("Error connecting to postgres", err)
	}
	defer db.Close()

	conn, err := async.Init()
	if err != nil {
		log.Fatal("Error connecting to rabbit", err)
	}
	defer async.Cleanup()

	chClient, err := clickhouse.Connect(false)
	if err != nil {
		log.Fatal("Error connecting to clickhouse", err)
	}
	defer chClient.Close()

	activityHistoryQueue := activity_history.Create()
	activityHistoryQueue.Conn = conn
	go activityHistoryQueue.Register(2)

	playersQueue := player_crawl.Create()
	playersQueue.Db = db
	playersQueue.Conn = conn
	go playersQueue.Register(4)

	activityCharactersQueue := character_fill.Create()
	activityCharactersQueue.Db = db
	activityCharactersQueue.Conn = conn
	go activityCharactersQueue.Register(4)

	pgcrsClickhouseQueue := pgcr.CreateClickhouseQueue()
	pgcrsClickhouseQueue.Db = db
	pgcrsClickhouseQueue.Conn = conn
	pgcrsClickhouseQueue.Clickhouse = chClient
	go pgcrsClickhouseQueue.Register(5)

	bonusPgcrsFetchQueue := bonus_pgcr.CreateFetchWorker()
	bonusPgcrsFetchQueue.Db = db
	bonusPgcrsFetchQueue.Conn = conn
	go bonusPgcrsFetchQueue.Register(10)

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
