package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	os.MkdirAll(path[:len(path)-len("/devbox.db")], 0755)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS traffic (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mirror TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		bytes_in INTEGER DEFAULT 0,
		bytes_out INTEGER DEFAULT 0,
		status INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS health_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mirror TEXT NOT NULL,
		status TEXT NOT NULL,
		error_msg TEXT DEFAULT '',
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_traffic_mirror ON traffic(mirror);
	CREATE INDEX IF NOT EXISTS idx_traffic_created ON traffic(created_at);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) RecordTraffic(mirror, method, path string, bytesIn, bytesOut, status int) error {
	_, err := s.db.Exec(
		"INSERT INTO traffic (mirror, method, path, bytes_in, bytes_out, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		mirror, method, path, bytesIn, bytesOut, status, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) RecordHealthCheck(mirror, status, errMsg string) error {
	_, err := s.db.Exec(
		"INSERT INTO health_checks (mirror, status, error_msg, checked_at) VALUES (?, ?, ?, ?)",
		mirror, status, errMsg, time.Now().Format(time.RFC3339),
	)
	return err
}

type TrafficSummary struct {
	Mirror    string
	Requests  int
	BytesIn   int64
	BytesOut  int64
}

func (s *Store) GetTrafficSummary(from, to time.Time) ([]TrafficSummary, error) {
	rows, err := s.db.Query(
		"SELECT mirror, COUNT(*), SUM(bytes_in), SUM(bytes_out) FROM traffic WHERE created_at BETWEEN ? AND ? GROUP BY mirror",
		from.Format(time.RFC3339), to.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TrafficSummary
	for rows.Next() {
		var ts TrafficSummary
		if err := rows.Scan(&ts.Mirror, &ts.Requests, &ts.BytesIn, &ts.BytesOut); err != nil {
			return nil, err
		}
		summaries = append(summaries, ts)
	}
	return summaries, nil
}

type TrafficHourly struct {
	Hour     string
	Mirror   string
	Requests int
	BytesOut int64
}

func (s *Store) GetTrafficHourly(from, to time.Time) ([]TrafficHourly, error) {
	rows, err := s.db.Query(
		"SELECT strftime('%Y-%m-%dT%H:00:00Z', created_at), mirror, COUNT(*), SUM(bytes_out) FROM traffic WHERE created_at BETWEEN ? AND ? GROUP BY 1, mirror ORDER BY 1",
		from.Format(time.RFC3339), to.Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TrafficHourly
	for rows.Next() {
		var th TrafficHourly
		if err := rows.Scan(&th.Hour, &th.Mirror, &th.Requests, &th.BytesOut); err != nil {
			return nil, err
		}
		result = append(result, th)
	}
	return result, nil
}

type TrafficLog struct {
	ID        int64
	Mirror    string
	Method    string
	Path      string
	BytesOut  int64
	Status    int
	CreatedAt string
}

func (s *Store) GetRecentTraffic(limit int) ([]TrafficLog, error) {
	rows, err := s.db.Query(
		"SELECT id, mirror, method, path, bytes_out, status, created_at FROM traffic ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []TrafficLog
	for rows.Next() {
		var tl TrafficLog
		if err := rows.Scan(&tl.ID, &tl.Mirror, &tl.Method, &tl.Path, &tl.BytesOut, &tl.Status, &tl.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, tl)
	}
	return logs, nil
}

type HealthStatus struct {
	Mirror    string
	Status    string
	ErrorMsg  string
	CheckedAt string
}

func (s *Store) GetLatestHealth() ([]HealthStatus, error) {
	rows, err := s.db.Query(
		"SELECT mirror, status, error_msg, checked_at FROM health_checks WHERE id IN (SELECT MAX(id) FROM health_checks GROUP BY mirror)",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []HealthStatus
	for rows.Next() {
		var hs HealthStatus
		if err := rows.Scan(&hs.Mirror, &hs.Status, &hs.ErrorMsg, &hs.CheckedAt); err != nil {
			return nil, err
		}
		statuses = append(statuses, hs)
	}
	return statuses, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}