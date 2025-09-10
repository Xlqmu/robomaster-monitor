package storage

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Article
type Article struct {
	ID        int64
	Title     string
	URL       string
	Author    string
	Category  string
	CreatedAt time.Time
	PostedAt  string
	Notified  bool
}

// InitDB
func InitDB(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS articles (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        url TEXT NOT NULL UNIQUE,
        author TEXT,
        category TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        posted_at TEXT,
        notified BOOLEAN DEFAULT 0
    )`)
	if err != nil {
		return err
	}

	log.Println("✅ 数据库初始化成功")
	return nil
}

// SaveArticle
func SaveArticle(article *Article) (int64, error) {
	result, err := db.Exec(
		"INSERT OR IGNORE INTO articles (title, url, author, category, posted_at, notified) VALUES (?, ?, ?, ?, ?, ?)",
		article.Title, article.URL, article.Author, article.Category, article.PostedAt, article.Notified,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// ArticleExists
func ArticleExists(url string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM articles WHERE url = ?", url).Scan(&count)
	return count > 0, err
}

// GetNewArticles
func GetNewArticles(limit int) ([]Article, error) {
	rows, err := db.Query(
		"SELECT id, title, url, author, category, created_at, posted_at, notified FROM articles WHERE notified = 0 ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.Title, &article.URL, &article.Author,
			&article.Category, &article.CreatedAt, &article.PostedAt, &article.Notified); err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}

	return articles, nil
}

// MarkAsNotified
func MarkAsNotified(id int64) error {
	_, err := db.Exec("UPDATE articles SET notified = 1 WHERE id = ?", id)
	return err
}

// Close
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
