package repository

import (
	"database/sql"
	"pausetobuye/internal/models"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Repository interface {
	GetItems(status string) ([]models.Item, error)
	GetItemByID(id int) (*models.Item, error)
	CreateItem(item *models.Item) error
	UpdateItemStatus(id int, status string) error
	GetConfig() (*models.Config, error)
	UpdateConfig(config *models.Config) error
	GetStats() (*models.Stats, error)
	Close() error
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.initDB(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *SQLiteRepository) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		price REAL NOT NULL,
		link TEXT,
		notes TEXT,
		category TEXT,
		wait_days INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		wait_until DATETIME NOT NULL,
		status TEXT DEFAULT 'waiting'
	);

	CREATE TABLE IF NOT EXISTS config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		hourly_wage REAL DEFAULT 15.0,
		ntfy_topic TEXT DEFAULT ''
	);

	INSERT OR IGNORE INTO config (id, hourly_wage) VALUES (1, 15.0);
	`
	
	_, err := r.db.Exec(schema)
	return err
}

func (r *SQLiteRepository) GetItems(status string) ([]models.Item, error) {
	var rows *sql.Rows
	var err error
	
	if status != "" {
		rows, err = r.db.Query(`
			SELECT id, title, price, link, notes, category, wait_days, created_at, wait_until, status
			FROM items WHERE status = ?
			ORDER BY created_at DESC
		`, status)
	} else {
		rows, err = r.db.Query(`
			SELECT id, title, price, link, notes, category, wait_days, created_at, wait_until, status
			FROM items
			ORDER BY created_at DESC
		`)
	}
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var items []models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Title, &item.Price, &item.Link, &item.Notes,
			&item.Category, &item.WaitDays, &item.CreatedAt, &item.WaitUntil, &item.Status)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	
	return items, nil
}

func (r *SQLiteRepository) GetItemByID(id int) (*models.Item, error) {
	var item models.Item
	err := r.db.QueryRow(`
		SELECT id, title, price, link, notes, category, wait_days, created_at, wait_until, status
		FROM items WHERE id = ?
	`, id).Scan(&item.ID, &item.Title, &item.Price, &item.Link, &item.Notes,
		&item.Category, &item.WaitDays, &item.CreatedAt, &item.WaitUntil, &item.Status)
	
	if err != nil {
		return nil, err
	}
	
	return &item, nil
}

func (r *SQLiteRepository) CreateItem(item *models.Item) error {
	waitUntil := time.Now().AddDate(0, 0, item.WaitDays)
	
	_, err := r.db.Exec(`
		INSERT INTO items (title, price, link, notes, category, wait_days, wait_until)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.Title, item.Price, item.Link, item.Notes, item.Category, item.WaitDays, waitUntil)
	
	return err
}

func (r *SQLiteRepository) UpdateItemStatus(id int, status string) error {
	_, err := r.db.Exec("UPDATE items SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *SQLiteRepository) GetConfig() (*models.Config, error) {
	var config models.Config
	err := r.db.QueryRow("SELECT hourly_wage, ntfy_topic FROM config WHERE id = 1").
		Scan(&config.HourlyWage, &config.NtfyTopic)
	
	if err != nil {
		return nil, err
	}
	
	return &config, nil
}

func (r *SQLiteRepository) UpdateConfig(config *models.Config) error {
	_, err := r.db.Exec("UPDATE config SET hourly_wage = ?, ntfy_topic = ? WHERE id = 1",
		config.HourlyWage, config.NtfyTopic)
	return err
}

func (r *SQLiteRepository) GetStats() (*models.Stats, error) {
	var stats models.Stats
	
	r.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&stats.TotalItems)
	r.db.QueryRow("SELECT COUNT(*) FROM items WHERE status = 'skipped'").Scan(&stats.ItemsSkipped)
	r.db.QueryRow("SELECT COUNT(*) FROM items WHERE status = 'bought'").Scan(&stats.ItemsBought)
	r.db.QueryRow("SELECT COALESCE(SUM(price), 0) FROM items WHERE status = 'skipped'").Scan(&stats.MoneySaved)
	r.db.QueryRow("SELECT COALESCE(SUM(price), 0) FROM items WHERE status = 'bought'").Scan(&stats.MoneySpent)
	
	var category sql.NullString
	r.db.QueryRow(`
		SELECT category FROM items 
		WHERE category != '' 
		GROUP BY category 
		ORDER BY COUNT(*) DESC 
		LIMIT 1
	`).Scan(&category)
	
	if category.Valid {
		stats.TopCategory = category.String
	} else {
		stats.TopCategory = "None"
	}
	
	return &stats, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}