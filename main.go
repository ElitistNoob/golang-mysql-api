package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"example.com/restapi/internal/api/books"
	"example.com/restapi/internal/db"
	"example.com/restapi/internal/web"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true", dbUsername, dbPassword, dbHost, dbPort, dbName)

	db, err := db.InitDatabase(dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database Initialized")
	defer db.Close()

	r := mux.NewRouter()

	booksHandler := books.NewHandler(db)
	webHandler := web.NewHandler()

	r.HandleFunc("/api/books", booksHandler.GetBooks).Methods("GET")
	r.HandleFunc("/api/books/{id}", booksHandler.GetBook).Methods("GET")
	r.HandleFunc("/api/books", booksHandler.CreateBook).Methods("POST")
	r.HandleFunc("/api/books/{id}", booksHandler.UpdateBook).Methods("PUT")
	r.HandleFunc("/api/books/{id}", booksHandler.DeleteBook).Methods("DELETE")
	r.HandleFunc("/", webHandler.Render)

	fs := http.FileServer(http.Dir("./dist"))
  r.PathPrefix("/dist/").Handler(http.StripPrefix("/dist/", fs))

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
