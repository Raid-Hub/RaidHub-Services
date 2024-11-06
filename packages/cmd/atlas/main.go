package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"raidhub/packages/monitoring"
	"raidhub/packages/postgres"
	"raidhub/packages/rabbit"
)

var (
	numWorkers       = flag.Int("workers", 50, "number of workers to spawn at the start")
	buffer           = flag.Int64("buffer", 10_000, "number of ids to start behind last added")
	targetInstanceId = flag.Int64("target", -1, "specific instance id to start at (optional)")
	workers          = 0
	periodLength     = 50_000
)

func main() {
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	workers = *numWorkers
	if *buffer < 0 || workers <= 0 || workers > maxWorkers {
		log.Fatalln("Invalid flags")
	}

	db, err := postgres.Connect()
	if err != nil {
		log.Fatalf("Error connecting to the database: %s", err)
	}
	defer db.Close()

	var instanceId int64
	if *targetInstanceId == -1 {
		instanceId, err = postgres.GetLatestInstanceId(db, *buffer)
		if err != nil {
			log.Fatalf("Error getting latest instance id: %s", err)
		}
	} else {
		instanceId = *targetInstanceId
	}

	monitoring.RegisterPrometheus(8080)

	run(instanceId, db)

}

func run(latestId int64, db *sql.DB) {
	defer func() {
		if r := recover(); r != nil {
			handlePanic(r)
		}
	}()

	conn, err := rabbit.Init()
	if err != nil {
		log.Fatalf("Failed to create connection: %s", err)
	}
	defer rabbit.Cleanup()

	rabbitChannel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %s", err)
	}
	defer rabbitChannel.Close()

	consumerConfig := ConsumerConfig{
		LatestId:       latestId,
		OffloadChannel: make(chan int64),
		RabbitChannel:  rabbitChannel,
	}

	sendStartUpAlert()

	// Start a goroutine to offload malformed or slowly resolving PGCRs
	go offloadWorker(consumerConfig.OffloadChannel, consumerConfig.RabbitChannel, db)

	for {
		workers = spawnWorkers(workers, db, &consumerConfig)
	}

}

func spawnWorkers(countWorkers int, db *sql.DB, consumerConfig *ConsumerConfig) int {
	var wg sync.WaitGroup
	ids := make(chan int64, 5)

	logWorkersStarting(countWorkers, periodLength, consumerConfig.LatestId)

	for i := 0; i < countWorkers; i++ {
		wg.Add(1)
		go Worker(&wg, ids, consumerConfig.OffloadChannel, consumerConfig.RabbitChannel, db)
	}

	// Pass IDs to workers
	for i := 0; i < periodLength; i++ {
		consumerConfig.LatestId++
		ids <- consumerConfig.LatestId
	}

	close(ids)
	wg.Wait()

	medianLag, err := execQuery(`histogram_quantile(0.20, sum(rate(pgcr_crawl_summary_lag_bucket[2m])) by (le))`, 3)
	if err != nil {
		log.Fatal(err)
	} else if medianLag == -1 {
		medianLag = 900
	}

	fractionNotFound, err := get404Fraction(4)
	if err != nil {
		log.Fatal(err)
	}

	logIntervalState(medianLag, countWorkers, fractionNotFound*100)

	newWorkers := 0
	if fractionNotFound == 0 {
		// how much we expect to get catch up
		periodLength = int(math.Round(math.Pow(float64(countWorkers)*(math.Ceil(medianLag)-20.0), 0.824)))
		if periodLength < 10_000 {
			periodLength = 10_000
		}
		// If we aren't getting 404's, just spike the workers up to ensure we catch up to live ASAP
		newWorkers = int(math.Ceil(float64(countWorkers) * (1 + float64(medianLag-20)/100)))

	} else {
		adjf := fractionNotFound - 0.025 // do not let workers go below 2.5 %
		decreaseFraction := math.Pow(retryDelayTime/8*math.Abs(adjf), 0.88) / 100
		if decreaseFraction > 0.65 {
			decreaseFraction = 0.65
		}
		sign := adjf / math.Abs(adjf)
		// Adjust number of workers for the next period
		newWorkers = int(math.Round(float64(countWorkers) - sign*decreaseFraction*float64(countWorkers)))
		periodLength = 1000 * newWorkers
	}

	if newWorkers > maxWorkers {
		newWorkers = maxWorkers
	} else if newWorkers < minWorkers {
		newWorkers = minWorkers
	}
	return newWorkers
}

func get404Fraction(intervalMins int) (float64, error) {
	f, err := execQuery(fmt.Sprintf(`sum(rate(pgcr_crawl_summary_status{status="3"}[%dm])) / sum(rate(pgcr_crawl_summary_status{}[%dm]))`, intervalMins, intervalMins), intervalMins)
	if err != nil || f == -1 {
		return 0, err
	} else {
		return f, err
	}
}

func execQuery(query string, intervalMins int) (float64, error) {
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", time.Now().Add(time.Duration(-intervalMins)*time.Minute).Format(time.RFC3339))
	params.Add("end", time.Now().Format(time.RFC3339))
	params.Add("step", "1m")

	client := http.Client{
		Timeout: time.Second * 10,
	}

	url := fmt.Sprintf("http://localhost:9090/api/v1/query_range?%s", params.Encode())

	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var res monitoring.QueryRangeResponse
	err = decoder.Decode(&res)
	if err != nil {
		return 0, err
	}

	// Creates a weighted average over the interval
	c := 0
	s := 0.0

	if len(res.Data.Result) == 0 {
		return -1, nil
	}

	for idx, y := range res.Data.Result[0].Values {
		val, err := strconv.ParseFloat(y[1].(string), 64)
		if err != nil {
			log.Fatal(err)
		}
		if math.IsNaN(val) {
			continue
		}
		c += (idx + 1)
		s += float64(idx+1) * val
	}

	if c == 0 {
		c = 1
	}

	return s / float64(c), nil
}
