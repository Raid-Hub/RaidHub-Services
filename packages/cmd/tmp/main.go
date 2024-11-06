package main

import (
	"encoding/json"
	"log"
	"raidhub/packages/async/pgcr_clickhouse"
	"raidhub/packages/bungie"
	"raidhub/packages/pgcr"
	"raidhub/packages/postgres"
	"raidhub/packages/rabbit"
)

func main() {
	conn, err := rabbit.Init()
	defer rabbit.Cleanup()
	if err != nil {
		panic(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	db, err := postgres.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query(`select data from pgcr join instance using (instance_id) where hash = 2192826039`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	log.Println("done querying")

	for rows.Next() {
		var data bungie.DestinyPostGameCarnageReport
		var bytes []byte
		err = rows.Scan(&bytes)
		if err != nil {
			panic(err)
		}

		// Decompress the JSON data
		decompressedJSON, err := pgcr.GzipDecompress(bytes)
		if err != nil {
			panic(err)
		}

		// Unmarshal the JSON back to a struct
		err = json.Unmarshal(decompressedJSON, &data)
		if err != nil {
			panic(err)
		}

		processed, err := pgcr.ProcessDestinyReport(&data)
		if err != nil {
			panic(err)
		}

		log.Println(processed.InstanceId)
		pgcr_clickhouse.SendToClickhouse(ch, processed)
	}
}
