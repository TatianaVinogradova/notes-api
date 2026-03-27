package main

import (
	"fmt"
	"log"
	"net/http"

	"notes-api/internal/repository"
	"notes-api/internal/service"
	"notes-api/internal/handler"
)

func main() {
	db, err := repository.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(nil)

	repo := repository.NewNoteRepository(db)
	svc := service.NewNoteService(repo)
	h := handler.NewNoteHandler(svc)

	http.Handle("/notes", h)
	http.Handle("/notes/", h)

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}