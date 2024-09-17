package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
	if db == nil {
        http.Error(w, "Database not initialized", http.StatusInternalServerError)
        return
    }

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	result := db.Create(&user)
	if result.Error != nil {
		// Log the error
		log.Printf("Error creating user: %v", result.Error)
		
		// Check for unique constraint violation
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed") {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		
		// Handle other errors
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	user.ID = idCounter
	users[idCounter] = user
	idCounter++
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var user User
	if db.First(&user, id).Error != nil {
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

	srv := &http.Server{
		Addr: ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("listen: %s \n", err)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<- quit
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5 *time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil{
		fmt.Println("Server forced to shutdown:", err)
	}
}