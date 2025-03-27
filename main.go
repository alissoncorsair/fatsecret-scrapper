package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alissoncorsair/fatsecret-scrapper/scraper"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	//http.HandleFunc("GET /api/scrape", scrapeHandler)
	http.HandleFunc("GET /api/users", getUsersHandler)
	http.HandleFunc("POST /api/users", addUserHandler)
	http.HandleFunc("GET /api/diary", getDiaryHandler)
	http.HandleFunc("GET /api/diary/{username}/{id}", getDiaryHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

/* func scrapeHandler(w http.ResponseWriter, r *http.Request) {
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
} */

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
	login := os.Getenv("FATSECRET_LOGIN")
	password := os.Getenv("FATSECRET_PASSWORD")
	username := r.PathValue("username")
	fmt.Println("username", username)
	id := r.PathValue("id")
	date := r.URL.Query().Get("date")

	var user = scraper.User{
		Username: username,
		ID:       id,
	}

	if date == "" {
		diaries := scraper.ScrapeFatSecret(login, password, []scraper.User{user})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(diaries)
		return
	}

	convertedDate, err := time.Parse("02/01/2006", date)
	fmt.Println("date", date)
	fmt.Println("convertedDate", convertedDate)
	if err != nil {
		http.Error(w, "Invalid date format. Use DD/MM/YYYY", http.StatusBadRequest)
		return
	}
	diaries := scraper.ScrapeFatSecret(login, password, []scraper.User{user}, convertedDate)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(diaries)
}
