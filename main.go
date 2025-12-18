package main

import (
	"log"
	"net/http"
	"teletubpax-api/routing"
)

func main() {
	router := routing.SetupRoutes()
	
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
