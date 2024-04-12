package main

import (
	"raidhub/shared/async/activity_history"
	"raidhub/shared/async/bonus_pgcr"
	"raidhub/shared/async/player_crawl"
	"raidhub/shared/monitoring"
)

func main() {
	go player_crawl.Register(3)
	go bonus_pgcr.Register(1)
	go activity_history.Register(4)

	monitoring.RegisterPrometheus(8083)

	// Keep the main thread running
	forever := make(chan bool)
	<-forever
}
