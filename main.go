package main

import (
	"log"
	"net/http"
	"os"

	"pausetobuye/internal/handlers"
	"pausetobuye/internal/repository"
)

func main() {
	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./pausetobuye.db"
	}

	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	// Initialize handlers
	h := handlers.NewHandler(repo)

	// Routes
	http.HandleFunc("/", h.HomeHandler)
	http.HandleFunc("/add", h.AddItemHandler)
	http.HandleFunc("/item/", h.ItemHandler)
	http.HandleFunc("/update-status/", h.UpdateStatusHandler)
	http.HandleFunc("/stats", h.StatsHandler)
	http.HandleFunc("/config", h.ConfigHandler)
	http.HandleFunc("/check-notifications", h.CheckNotificationsHandler)

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("PauseToBuye server running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}