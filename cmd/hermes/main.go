package main

import (
	"raidhub/shared/async"
	"raidhub/shared/async/activity_history"
	"raidhub/shared/async/bonus_pgcr"
	"raidhub/shared/async/character_fill"
	"raidhub/shared/async/player_crawl"
	"raidhub/shared/monitoring"
	"raidhub/shared/postgres"
)

func main() {
	db, err := postgres.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	conn, err := async.Init()
	if err != nil {
		panic(err)
	}
	defer async.Cleanup()

	activityHistoryQueue := activity_history.Create()
	activityHistoryQueue.Conn = conn
	go activityHistoryQueue.Register(2)

	playersQueue := player_crawl.Create()
	playersQueue.Db = db
	playersQueue.Conn = conn
	go playersQueue.Register(5)

	pgcrsQueue := bonus_pgcr.Create()
	pgcrsQueue.Db = db
	pgcrsQueue.Conn = conn
	go pgcrsQueue.Register(5)

	activityCharactersQueue := character_fill.Create()
	activityCharactersQueue.Db = db
	activityCharactersQueue.Conn = conn
	go activityCharactersQueue.Register(4)

	monitoring.RegisterPrometheus(8083)

	// Keep the main thread running
	forever := make(chan bool)
	<-forever
}
