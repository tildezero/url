package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v4"
)

func main() {

	db, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln("error connecting to postgres:", err)
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("/{slug}"))
	})
	r.Post("/shorten", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if !(q.Has("short") && q.Has("long") && q.Has("password")) {
			http.Error(w, "/shorten?short=shorturl&long=longurl", http.StatusNotFound)
			return
		}

		if q.Get("password") != os.Getenv("PASSWORD") {
			log.Println("wrong password given:", q.Get("password"), "instead of", os.Getenv("PASSWORD"))
			http.Error(w, "wrong password", http.StatusUnauthorized)
			return
		}

		u, err := url.Parse(q.Get("long"))
		if err != nil || u.Scheme == "" {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}
		_, err = db.Exec(context.Background(), "insert into links values ($1, $2)", q.Get("short"), q.Get("long"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("url shortened"))

	})
	r.Get("/{slug}", func(w http.ResponseWriter, r *http.Request) {
		s := chi.URLParam(r, "slug")
		var url sql.NullString

		err := db.QueryRow(r.Context(), "select url from links where slug=$1", s).Scan(&url)
		if err != nil {
			if err == pgx.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				http.Error(w, "404 page not found", http.StatusNotFound)
				return
			} else {
				http.Error(w, "error"+err.Error(), http.StatusInternalServerError)
			}
		}

		if url.Valid {
			http.Redirect(w, r, url.String, http.StatusMovedPermanently)
		} else {
			http.Error(w, "idk", http.StatusInternalServerError)
			return
		}

	})
	url := ":" + os.Getenv("PORT")
	http.ListenAndServe(url, r)
}
