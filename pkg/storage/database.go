package storage

import "database/sql"

type LastUserQuery struct {
	Word     string
	Type     string
	Language string
}

func UpdateUserLanguage(db *sql.DB, userID int, language string) error {
	// SQL query for upsert operation
	query := `
    INSERT INTO users (id, language, help_type)
    VALUES (?, ?, ?)
    ON CONFLICT(id)
    DO UPDATE SET language = EXCLUDED.language;
    `
	_, err := db.Exec(query, userID, language, "")
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

func UpdateUserHelpType(db *sql.DB, userID int, helpType string) error {
	query := `
    UPDATE users SET help_type = ?
    WHERE id = ?;
    `
	_, err := db.Exec(query, helpType, userID)
	if err != nil {
		return err
	}
	return nil
}

func GetUserHelpType(db *sql.DB, userID int) (string, error) {
	query := `
	SELECT help_type FROM users WHERE id = ?;
	`
	var helpType string
	err := db.QueryRow(query, userID).Scan(&helpType)
	if err != nil {
		return "", err
	}
	return helpType, nil
}

func StoreQuery(db *sql.DB, userID int, helpType, language, word string) (int, error) {
	query := `
	INSERT INTO queries (user_id, help_type, language, word)
	VALUES (?, ?, ?, ?)
	RETURNING id;
	`
	var queryID int
	err := db.QueryRow(query, userID, helpType, language, word).Scan(&queryID)
	if err != nil {
		return 0, err
	}

	return queryID, nil
}

func GetLastUserQuery(db *sql.DB, userID int) (*LastUserQuery, error) {
	// select last query from user, join with cached_responses to get type
	query := `
  SELECT q.word, q.help_type, q.language
  FROM queries q
  WHERE q.user_id = ?
  ORDER BY q.timestamp DESC
  LIMIT 1;
  `
	var lastQuery LastUserQuery
	qr := db.QueryRow(query, userID)
	if qr.Err() != nil {
		return nil, qr.Err()
	}
	qr.Scan(&lastQuery.Word, &lastQuery.Type, &lastQuery.Language)
	return &lastQuery, nil
}

func CacheResponse(db *sql.DB, query_id int, response string) error {
	query := `
  INSERT INTO cached_responses (query_id, response)
  VALUES (?, ?);
  `
	_, err := db.Exec(query, query_id, response)
	if err != nil {
		return err
	}
	return nil
}

func GetCachedResponseByWordLangAndType(db *sql.DB, language, helpType, word string) (string, error) {
	query := `
  SELECT cr.response
  FROM cached_responses cr
  JOIN queries q ON q.id = cr.query_id
  WHERE q.language = ? AND q.help_type = ? AND q.word = ? ;
  `
	var response string
	qr := db.QueryRow(query, language, helpType, word)
	err := qr.Err()
	if err != nil {
		return "", err
	}
	qr.Scan(&response)
	return response, nil
}

func CleanOldCachedResponses(db *sql.DB) error {
	query := `
        DELETE FROM cached_responses
        WHERE datetime(created_at) < datetime('now', '-24 hours');
    `
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}
