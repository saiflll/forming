package main

import (
	"time"
)

// Get skip logs
func getSkipLogs() ([]map[string]interface{}, error) {
	if db == nil {
		return []map[string]interface{}{}, nil
	}

	query := `SELECT prefix, ts, reg2, reg5, reg114, reason, skipped_at 
	          FROM skip_log 
	          ORDER BY skipped_at DESC 
	          LIMIT 100`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var prefix, ts, reason string
		var reg2, reg5, reg114 int
		var skippedAt time.Time

		if err := rows.Scan(&prefix, &ts, &reg2, &reg5, &reg114, &reason, &skippedAt); err != nil {
			return nil, err
		}

		logs = append(logs, map[string]interface{}{
			"prefix":     prefix,
			"ts":         ts,
			"reg2":       reg2,
			"reg5":       reg5,
			"reg114":     reg114,
			"reason":     reason,
			"skipped_at": skippedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return logs, nil
}
