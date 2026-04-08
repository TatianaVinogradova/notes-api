package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"notes-api/internal/handler"
	"notes-api/internal/repository"
	"notes-api/internal/service"
)

func main() {
	db, err := repository.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.TODO())

	repo := repository.NewPostgresRepository(db)
	svc := service.NewNoteService(repo)

	webHandler, err := handler.NewWebHandler(svc)
	if err != nil {
		log.Fatal(err)
	}

	apiHandler := handler.NewNoteHandler(svc)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/web" {
			webHandler.Index(w, r)
			return
		}
		if path == "/web/notes" {
			webHandler.Create(w, r)
			return
		}
		if strings.HasPrefix(path, "/web/notes/") {
			switch {
			case strings.HasSuffix(path, "/delete"):
				webHandler.Delete(w, r)
			case strings.HasSuffix(path, "/edit"):
				if r.Method == http.MethodGet {
					webHandler.Edit(w, r)
				} else {
					webHandler.Update(w, r)
				}
			default:
				http.NotFound(w, r)
			}
			return
		}

		if path == "/notes" || path == "/notes/" || strings.HasPrefix(path, "/notes/") {
			apiHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	fmt.Println("Server running on http://localhost:8080")
	fmt.Println("Frontend en http://localhost:8080/web")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
