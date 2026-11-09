package helper

import (
	"database/sql"
	"fmt"
)

func IsSqliteTableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?;`
	err := db.QueryRow(query, tableName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func SqliteOpen(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sql.Open[%s] 失败: %w", dbPath, err)
	}

	// real check
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sql.Ping 失败: %s: %w", dbPath, err)
	}

	return db, nil
}
