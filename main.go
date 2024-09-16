package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
    fmt.Println("Serving static files on :8080...")
    http.ListenAndServe(":8080", nil)
}