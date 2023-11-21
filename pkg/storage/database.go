package storage

import "database/sql"

func UpdateUserLanguage(db *sql.DB, userID int, language string) error {
	// SQL query for upsert operation
	query := `
    INSERT INTO users (id, language)
    VALUES (?, ?)
    ON CONFLICT(id)
    DO UPDATE SET language = EXCLUDED.language;
    `
	_, err := db.Exec(query, userID, language)
	if err != nil {
		return err
	}
	return nil
}

func GetUserLanguage(db *sql.DB, userID int) (string, error) {
	query := `
	SELECT language FROM users WHERE id = ?;
	`
	var language string
	err := db.QueryRow(query, userID).Scan(&language)
	if err != nil {
		return "", err
	}
	return language, nil
}
