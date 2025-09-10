package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Post struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {

	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL не задан (в .env или переменных окружения)")
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":9005"
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("pgxpool.ParseConfig: %v", err)
	}

	cfg.MaxConns = 5
	cfg.MinConns = 0

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("pgxpool.NewWithConfig: %v", err)
	}
	defer pool.Close()

	{
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			log.Fatalf("Не удалось подключиться к БД: %v", err)
		}
	}
	log.Println("✅ Подключение к PostgreSQL установлено")

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.GET("/posts", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		rows, err := pool.Query(ctx, `
			SELECT id, title, body, created_at
			FROM posts
			ORDER BY id DESC
			LIMIT 100
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query error", "detail": err.Error()})
			return
		}
		defer rows.Close()

		posts := make([]Post, 0, 16)
		for rows.Next() {
			var p Post
			if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error", "detail": err.Error()})
				return
			}
			posts = append(posts, p)
		}
		if rows.Err() != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rows error", "detail": rows.Err().Error()})
			return
		}

		c.JSON(http.StatusOK, posts)
	})

	go func() {
		if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gin Run: %v", err)
		}
	}()
	log.Printf("🚀 Сервер запущен на %s", addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("🛑 Завершение работы")
}
