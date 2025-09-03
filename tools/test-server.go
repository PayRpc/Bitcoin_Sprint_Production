package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "test server working", "port": "8080"}`)
	})

	fmt.Println("Test server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
