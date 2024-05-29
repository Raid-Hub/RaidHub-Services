package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickhouseInstance = map[string]interface{}

func InsertProcessedInstances(conn driver.Conn, instances []ClickhouseInstance) error {
	ctx := context.Background()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO instance SETTINGS async_insert=1, wait_for_async_insert=1")
	if err != nil {
		return fmt.Errorf("error preparing batch for instances: %s", err)
	}

	for _, instance := range instances {
		err = batch.Append(
			instance["instance_id"],
			instance["hash"],
			instance["completed"],
			instance["player_count"],
			instance["fresh"],
			instance["flawless"],
			instance["date_started"],
			instance["date_completed"],
			instance["platform_type"],
			instance["duration"],
			instance["score"],
			instance["players"])
		if err != nil {
			return err
		}
	}

	return batch.Send()
}
