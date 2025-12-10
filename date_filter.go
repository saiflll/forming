package main

import (
	"database/sql"
	"fmt"
)

// getRecordsByDateRange retrieves records filtered by date range
func getRecordsByDateRange(startDate, endDate, prefixFilter, statusFilter, sortBy string) ([]Record, error) {
	if db == nil {
		return []Record{}, nil
	}

	query := "SELECT id, ts, reg2, reg5, reg114, prefix, created_at FROM production_mdcw WHERE 1=1"
	args := []interface{}{}
	argId := 1

	// Date range filter
	if startDate != "" && endDate != "" {
		query += fmt.Sprintf(" AND created_at >= $%d AND created_at < ($%d::date + interval '1 day')", argId, argId+1)
		args = append(args, startDate, endDate)
		argId += 2
	} else if startDate != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argId)
		args = append(args, startDate)
		argId++
	} else if endDate != "" {
		query += fmt.Sprintf(" AND created_at < ($%d::date + interval '1 day')", argId)
		args = append(args, endDate)
		argId++
	}

	// Filter by prefix (optional)
	if prefixFilter != "" && prefixFilter != "all" {
		query += fmt.Sprintf(" AND prefix = $%d", argId)
		args = append(args, prefixFilter)
		argId++
	}

	// Filter by status
	if statusFilter != "" && statusFilter != "all" {
		query += fmt.Sprintf(" AND reg5 = $%d", argId)
		args = append(args, statusFilter)
		argId++
	}

	// Sorting
	if sortBy == "weight_desc" {
		query += " ORDER BY reg114 DESC"
	} else if sortBy == "weight_asc" {
		query += " ORDER BY reg114 ASC"
	} else {
		query += " ORDER BY created_at DESC"
	}

	query += " LIMIT 1000" // Allow more results for custom date range

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		var prefix sql.NullString
		var reg2, reg5, reg114 sql.NullInt64

		if err := rows.Scan(&r.ID, &r.Ts, &reg2, &reg5, &reg114, &prefix, &r.CreatedAt); err != nil {
			return nil, err
		}

		if prefix.Valid {
			r.Prefix = prefix.String
		} else {
			r.Prefix = "-"
		}

		if reg2.Valid {
			r.Reg2 = int(reg2.Int64)
		}
		if reg5.Valid {
			r.Reg5 = int(reg5.Int64)
		}
		if reg114.Valid {
			r.Reg114 = int(reg114.Int64)
		}

		records = append(records, r)
	}
	return records, nil
}

// getPrefixes returns unique prefixes from database
func getPrefixes() ([]string, error) {
	if db == nil {
		return []string{}, nil
	}

	query := `SELECT DISTINCT prefix FROM production_mdcw WHERE prefix IS NOT NULL ORDER BY prefix ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefixes []string
	for rows.Next() {
		var prefix string
		if err := rows.Scan(&prefix); err != nil {
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}
