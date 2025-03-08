package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type FoodItem struct {
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
	Fat      string `json:"fat"`
	Carbs    string `json:"carbs"`
	Protein  string `json:"protein"`
	Calories string `json:"calories"`
}

type MealData struct {
	Name     string     `json:"name"`
	Fat      string     `json:"fat"`
	Carbs    string     `json:"carbs"`
	Protein  string     `json:"protein"`
	Calories string     `json:"calories"`
	Items    []FoodItem `json:"items"`
}

type DiaryEntry struct {
	Date      string     `json:"date"`
	Calories  string     `json:"calories"`
	IDR       string     `json:"idr"`
	Fat       string     `json:"fat"`
	Protein   string     `json:"protein"`
	Carbs     string     `json:"carbs"`
	Timestamp string     `json:"timestamp"`
	Meals     []MealData `json:"meals"`
}

type User struct {
	Username string `json:"username"`
	ID       string `json:"id"`
}

const (
	baseURL         = "https://www.fatsecret.com.br"
	loginPageURL    = "https://www.fatsecret.com.br/Auth.aspx?pa=s"
	OutputDir       = "output"
	ConfigDir       = "config"
	UsersConfigFile = "users.json"
)

func LoadUsers() ([]User, error) {
	configPath := filepath.Join(ConfigDir, UsersConfigFile)

	if _, err := os.Stat(ConfigDir); os.IsNotExist(err) {
		if err := os.MkdirAll(ConfigDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %v", err)
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		initialData := []User{
			{Username: "alissoncorsair", ID: "77829510"},
		}
		jsonData, err := json.MarshalIndent(initialData, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal initial users data: %v", err)
		}

		if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write initial users config: %v", err)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users config: %v", err)
	}

	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users config: %v", err)
	}

	return users, nil
}

func SaveUsers(users []User) error {
	configPath := filepath.Join(ConfigDir, UsersConfigFile)

	if _, err := os.Stat(ConfigDir); os.IsNotExist(err) {
		if err := os.MkdirAll(ConfigDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
	}

	jsonData, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %v", err)
	}

	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write users config: %v", err)
	}

	fmt.Printf("Updated users configuration at %s\n", configPath)
	return nil
}

func saveUserDataToJSON(user User, entry DiaryEntry) error {
	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	entry.Timestamp = time.Now().Format("02/01/2006")

	data := struct {
		User  User       `json:"user"`
		Entry DiaryEntry `json:"entry"`
	}{
		User:  user,
		Entry: entry,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %v", user.Username, err)
	}

	filename := filepath.Join(OutputDir, fmt.Sprintf("%s_%s.json",
		user.Username,
		time.Now().Format("02-01-2006")))

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file for %s: %v", user.Username, err)
	}

	fmt.Printf("Saved data for %s to %s\n", user.Username, filename)
	return nil
}

func extractFormData(doc *goquery.Document) url.Values {
	formData := url.Values{}

	doc.Find("input[type='hidden']").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("value")
		if name != "" {
			formData.Add(name, value)
		}
	})

	return formData
}

func findLoginButtonID(doc *goquery.Document) string {
	var loginButtonID string

	doc.Find("button.signIn").Each(func(_ int, s *goquery.Selection) {
		onclick, exists := s.Attr("onclick")
		if exists && strings.Contains(onclick, "__doPostBack") {
			parts := strings.Split(onclick, "'")
			if len(parts) >= 2 {
				loginButtonID = parts[1]
				fmt.Println("Found login button ID:", loginButtonID)
			}
		}
	})

	if loginButtonID == "" {
		return "ctl00$ctl12$Logincontrol1$LoginButton"
	}
	return loginButtonID
}

func createLoginRequest(formData url.Values) (*http.Request, error) {
	req, err := http.NewRequest("POST", loginPageURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", loginPageURL)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	return req, nil
}

func extractFoodItem(tr *goquery.Selection) FoodItem {
	item := FoodItem{}

	nameLink := tr.Find("td:nth-child(1) a")
	if nameLink.Length() > 0 {
		item.Name = strings.TrimSpace(nameLink.Text())
	}

	quantityDiv := tr.Find("td:nth-child(1) div.smallText")
	if quantityDiv.Length() > 0 {
		item.Quantity = strings.TrimSpace(quantityDiv.Text())
	}

	tds := tr.Find("td.normal")
	if tds.Length() >= 1 {
		item.Fat = strings.TrimSpace(tds.Eq(0).Text())
	}

	if tds.Length() >= 2 {
		item.Carbs = strings.TrimSpace(tds.Eq(1).Text())
	}

	if tds.Length() >= 3 {
		item.Protein = strings.TrimSpace(tds.Eq(2).Text())
	}

	if tds.Length() >= 4 {
		item.Calories = strings.TrimSpace(tds.Eq(3).Text())
	}

	return item
}

func extractMealData(doc *goquery.Selection) MealData {
	meal := MealData{}

	headerRow := doc.Find("tr:first-child td table.foodsNutritionTbl tr:first-child")
	meal.Name = strings.TrimSpace(headerRow.Find("td.greytitlex").Text())

	tdSubs := headerRow.Find("td.sub")
	if tdSubs.Length() >= 1 {
		meal.Fat = strings.TrimSpace(tdSubs.Eq(0).Text())
	}

	if tdSubs.Length() >= 2 {
		meal.Carbs = strings.TrimSpace(tdSubs.Eq(1).Text())
	}

	if tdSubs.Length() >= 3 {
		meal.Protein = strings.TrimSpace(tdSubs.Eq(2).Text())
	}

	if tdSubs.Length() >= 4 {
		meal.Calories = strings.TrimSpace(tdSubs.Eq(3).Text())
	}

	foodItems := []FoodItem{}
	doc.Find("tr td.borderLeft.borderRight").Each(func(_ int, s *goquery.Selection) {
		foodItemTr := s.Find("table.foodsNutritionTbl tr")
		if foodItemTr.Length() > 0 {
			foodItem := extractFoodItem(foodItemTr)
			if foodItem.Name != "" {
				foodItems = append(foodItems, foodItem)
			}
		}
	})

	meal.Items = foodItems
	return meal
}

func PrintDiaryEntry(username string, entry DiaryEntry) {
	fmt.Printf("\n----- Most recent entry for %s (%s) -----\n", username, entry.Date)
	fmt.Printf("Calories: %s\n", entry.Calories)
	fmt.Printf("IDR: %s\n", entry.IDR)
	fmt.Printf("Fat: %s\n", entry.Fat)
	fmt.Printf("Protein: %s\n", entry.Protein)
	fmt.Printf("Carbs: %s\n", entry.Carbs)
}

func extractDetailedDiaryEntry(doc *goquery.Document) DiaryEntry {
	entry := DiaryEntry{}

	headerTable := doc.Find("div.MyFSHeaderFooterAdditional table.foodsNutritionTbl").First()
	if headerTable.Length() > 0 {
		tds := headerTable.Find("tr:nth-child(3) td.sub")
		if tds.Length() >= 1 {
			entry.Fat = strings.TrimSpace(tds.Eq(0).Text())
		}
		if tds.Length() >= 2 {
			entry.Carbs = strings.TrimSpace(tds.Eq(1).Text())
		}
		if tds.Length() >= 3 {
			entry.Protein = strings.TrimSpace(tds.Eq(2).Text())
		}
		if tds.Length() >= 4 {
			entry.Calories = strings.TrimSpace(tds.Eq(3).Text())
		}
	}

	dateText := doc.Find("div.subtitle").Text()
	if dateText != "" {
		entry.Date = strings.TrimSpace(dateText)
	}

	idrText := doc.Find("div.big").First().Text()
	if idrText != "" {
		entry.IDR = strings.TrimSpace(idrText)
	}

	meals := []MealData{}
	doc.Find("table.generic.foodsTbl").Each(func(i int, s *goquery.Selection) {
		meal := extractMealData(s)
		meals = append(meals, meal)
	})

	entry.Meals = meals

	return entry
}

func getUserDiaryEntry(client *http.Client, user User) (DiaryEntry, error) {
	foodDiaryURL := fmt.Sprintf("https://www.fatsecret.com.br/Diary.aspx?pa=fj&id=%s", user.ID)
	fmt.Printf("Accessing food journal for %s...\n", user.Username)

	foodDiaryResp, err := client.Get(foodDiaryURL)
	if err != nil {
		return DiaryEntry{}, err
	}
	defer foodDiaryResp.Body.Close()

	foodDiaryDoc, err := goquery.NewDocumentFromReader(foodDiaryResp.Body)
	if err != nil {
		return DiaryEntry{}, err
	}

	detailedEntry := extractDetailedDiaryEntry(foodDiaryDoc)

	fmt.Printf("\n----- Food diary for %s (%s) -----\n", user.Username, detailedEntry.Date)
	fmt.Printf("Calories: %s\n", detailedEntry.Calories)
	fmt.Printf("IDR: %s\n", detailedEntry.IDR)
	fmt.Printf("Fat: %s g\n", detailedEntry.Fat)
	fmt.Printf("Protein: %s g\n", detailedEntry.Protein)
	fmt.Printf("Carbs: %s g\n", detailedEntry.Carbs)

	fmt.Println("\nMeal summary:")
	for _, meal := range detailedEntry.Meals {
		fmt.Printf("- %s: %s cal, %d items\n", meal.Name, meal.Calories, len(meal.Items))
	}

	return detailedEntry, nil
}

func ScrapeFatSecret(username, password string) map[string]DiaryEntry {
	users, err := LoadUsers()
	if err != nil {
		log.Fatalf("Error loading users config: %v", err)
	}

	if len(users) == 0 {
		log.Fatalf("No users found in configuration")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(loginPageURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println("Initial page status code:", resp.StatusCode)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	formData := extractFormData(doc)
	loginButtonID := findLoginButtonID(doc)

	formData.Add("ctl00$ctl12$Logincontrol1$Name", username)
	formData.Add("ctl00$ctl12$Logincontrol1$Password", password)
	formData.Add("ctl00$ctl12$Logincontrol1$CreatePersistentCookie", "on")

	formData.Set("__EVENTTARGET", loginButtonID)
	formData.Set("__EVENTARGUMENT", "")

	loginReq, err := createLoginRequest(formData)
	if err != nil {
		log.Fatal(err)
	}

	loginResp, err := client.Do(loginReq)
	if err != nil {
		log.Fatal(err)
	}
	defer loginResp.Body.Close()

	fmt.Println("Login response status code:", loginResp.StatusCode)
	fmt.Println("Response URL:", loginResp.Request.URL.String())

	userEntries := make(map[string]DiaryEntry)

	if loginResp.StatusCode == 302 {
		redirectURL := loginResp.Header.Get("Location")
		fmt.Println("Redirect URL:", redirectURL)

		if !strings.HasPrefix(redirectURL, "http") {
			redirectURL = baseURL + redirectURL
		}
		fmt.Println("Full redirect URL:", redirectURL)

		nextResp, err := client.Get(redirectURL)
		if err != nil {
			log.Fatal(err)
		}
		defer nextResp.Body.Close()

		fmt.Println("Redirect response status code:", nextResp.StatusCode)
		fmt.Println("After redirect URL:", nextResp.Request.URL.String())

		followClient := &http.Client{
			Jar: jar,
		}

		fmt.Println("\nAccessing food diary pages...")
		for _, user := range users {
			entry, err := getUserDiaryEntry(followClient, user)
			if err != nil {
				fmt.Printf("Error getting diary for %s: %v\n", user.Username, err)
				continue
			}

			if entry.Date != "" {
				userEntries[user.Username] = entry

				if err := saveUserDataToJSON(user, entry); err != nil {
					fmt.Println(err)
				}
			}
		}

		fmt.Println("\nLogin and data extraction successful!")
	} else {
		fmt.Println("Login failed - no redirect detected")
	}

	return userEntries
}

func RunScraper(username, password string) {
	entries := ScrapeFatSecret(username, password)

	fmt.Printf("\nSummary: Retrieved entries for %d users\n", len(entries))
	fmt.Printf("JSON files saved in the '%s' directory\n", OutputDir)
}
