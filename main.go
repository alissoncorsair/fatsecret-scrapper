package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alissoncorsair/fatsecret-scrapper/scraper"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	http.HandleFunc("GET /api/scrape", scrapeHandler)
	http.HandleFunc("GET /api/users", getUsersHandler)
	http.HandleFunc("POST /api/users", addUserHandler)
	http.HandleFunc("GET /api/diary/{username}", getDiaryHandler)
	http.HandleFunc("GET /api/diary/{username}/{date}", getDiaryHandler) //DD/MM/YYYY

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	username := os.Getenv("FATSECRET_LOGIN")
	password := os.Getenv("FATSECRET_PASSWORD")

	if username == "" || password == "" {
		http.Error(w, "FatSecret credentials not configured", http.StatusInternalServerError)
		return
	}

	entries := scraper.ScrapeFatSecret(username, password)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Scraped data for %d users", len(entries)),
		"count":   len(entries),
	})
}

func addUserHandler(w http.ResponseWriter, r *http.Request) {
	var newUser scraper.User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if newUser.Username == "" || newUser.ID == "" {
		http.Error(w, "Username and ID are required", http.StatusBadRequest)
		return
	}

	users, err := scraper.LoadUsers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading users: %v", err), http.StatusInternalServerError)
		return
	}

	for _, user := range users {
		if user.Username == newUser.Username {
			http.Error(w, "User with this username already exists", http.StatusConflict)
			return
		}
	}

	users = append(users, newUser)

	if err := scraper.SaveUsers(users); err != nil {
		http.Error(w, fmt.Sprintf("Error saving users: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := scraper.LoadUsers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading users: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)
}

func getDiaryHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	date := r.URL.Query().Get("date")

	if date == "" {
		files, err := filepath.Glob(filepath.Join(scraper.OutputDir, username+"_*.json"))
		if err != nil {
			http.Error(w, "Error reading diaries", http.StatusInternalServerError)
			return
		}

		if len(files) == 0 {
			http.Error(w, "No diary entries found for this user", http.StatusNotFound)
			return
		}

		latest := files[0]
		for _, file := range files {
			if file > latest {
				latest = file
			}
		}

		data, err := os.ReadFile(latest)
		if err != nil {
			http.Error(w, "Error reading diary", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	filename := filepath.Join(scraper.OutputDir, fmt.Sprintf("%s_%s.json", username, date))
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		http.Error(w, "Diary entry not found", http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		http.Error(w, "Error reading diary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
