package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDB() {
	var err error

	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(&User{})
}



type User struct {
	ID int `json:"id"`
	Name string `json:"name"`
}

var users = map[int]User{}
var idCounter = 1

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	user.ID = idCounter
	users[idCounter] = user
	idCounter++
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	user, ok := users[id]
	if !ok {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(user)
}

// Middleware

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Request: %s %s \n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		auth := r.Header.Get("Authorization")

		if auth != "Bearer-mytoken" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var jobQueue = make(chan string, 100)

func worker() {
	for job := range jobQueue {
		fmt.Println("Processing Job>:", job)
		time.Sleep(2 * time.Second)
	}
}

func enqueueJob(w http.ResponseWriter, r *http.Request) {
	job :=r.URL.Query().Get("job")

	jobQueue <- job

	fmt.Fprintln(w, "job added", job)
}

func main() {
	initDB()

	r := chi.NewRouter()

	r.Use(Logger)
	r.Use(authenticationMiddleware)

	r.Post("/users", createUser)
	r.Get("/users/{id}", getUser)

	r.Get("/enqueue", enqueueJob)

	for i := 0; i < 3; i++ {
		go worker()
	}

	fmt.Println("Server starting on :8080...")
	http.ListenAndServe(":8080", r)
}