package postgres

import (
	"database/sql"
)

func GetLatestInstanceId(db *sql.DB, buffer int64) (int64, error) {
	var latestID int64
	err := db.QueryRow(`SELECT instance_id FROM activity WHERE instance_id < 1000000000000 ORDER BY instance_id DESC LIMIT 1`).Scan(&latestID)
	if err != nil {
		return 0, err
	} else {
		return latestID - buffer, nil
	}
}
