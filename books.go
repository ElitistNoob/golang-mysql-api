package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Book struct {
	Id        int       `json:"id"`
	ISBN      string    `json:"isbn"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Publisher string    `json:"publisher"`
	CreatedAt time.Time `json:"createdAt"`
}

type Books = []Book

type Data struct {
	Books Books
}

var db *sql.DB
var tmpl *template.Template

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

	var err error
	db, err = initDatabase(dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database Initialized")
	defer db.Close()

	r := mux.NewRouter()

	r.HandleFunc("/api/books", getBooks).Methods("GET")
	r.HandleFunc("/api/books/{id}", getBook).Methods("GET")
	r.HandleFunc("/api/books", createBook).Methods("POST")
	r.HandleFunc("/api/books/{id}", updateBook).Methods("PUT")
	r.HandleFunc("/api/books/{id}", deleteBook).Methods("DELETE")
	r.HandleFunc("/", render).Methods("GET")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

func initDatabase(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

func render(w http.ResponseWriter, r *http.Request) {
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}

func getBooks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT * FROM books`)
	if err != nil {
		http.Error(w, "Failed to execute the database query", http.StatusInternalServerError)
		log.Printf("Failed to execute the database query: %v", err)
		return
	}
	defer rows.Close()

	var books Books
	for rows.Next() {
		var book Book
		if err := rows.Scan(&book.Id, &book.ISBN, &book.Title, &book.Author, &book.Publisher, &book.CreatedAt); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			log.Printf("Failed to scan row: %v", err)
			return
		}
		books = append(books, book)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error occured during row iteration", http.StatusInternalServerError)
		log.Printf("Error occured during row iteration: %v", err)
		return
	}

	data := Data{
		Books: books,
	}

	if err := tmpl.ExecuteTemplate(w, "book-list", data); err != nil {
		log.Fatal(err)
	}
}

func getBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "No matching param in url found", http.StatusBadRequest)
		fmt.Printf("No matching param in url found: %v", err)
		return
	}

	result := db.QueryRow("SELECT * FROM books WHERE Id LIKE ?", id)
	var book Book
	if err := result.Scan(&book.Id, &book.ISBN, &book.Title, &book.Author, &book.Publisher, &book.CreatedAt); err != nil {
		http.Error(w, "Query does not exist in Database", http.StatusInternalServerError)
		fmt.Printf("Quest does not exists in Database: %s", err)
		return
	}

	if err := json.NewEncoder(w).Encode(book); err != nil {
		http.Error(w, "Failed to encode book to JSON", http.StatusInternalServerError)
		log.Println("Failed to encode book to JSON: err")
		return
	}
}

func createBook(w http.ResponseWriter, r *http.Request) {
	isbn := r.FormValue("isbn")
	title := r.FormValue("title")
	author := r.FormValue("author")
	publisher := r.FormValue("publisher")
	createdAt := time.Now()

	result, err := db.Exec(`INSERT INTO books (isbn, title, author, publisher, created_at) VALUES (?,?,?,?,?)`,
		isbn, title, author, publisher, createdAt)
	if err != nil {
		http.Error(w, "Error inserting record", http.StatusInternalServerError)
		log.Printf("Error inserting record: %v", err)
		return
	}

	bookId, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Error retrieving last insert ID", http.StatusInternalServerError)
		log.Printf("Error retrieving last insert Id: %v", err)
		return
	}

	log.Printf("Book: %v, successfully created", bookId)
}

func updateBook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "No matching param found", http.StatusBadRequest)
		log.Printf("No matching param found: %v", err)
		return
	}

	var book Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		http.Error(w, "Invalid Request Payload", http.StatusBadRequest)
		log.Printf("Invalid Request Payload: %v", err)
		return
	}

	result, err := db.Exec(`UPDATE books SET publisher = ? WHERE id LIKE ?`, book.Publisher, id)
	if err != nil {
		http.Error(w, "Error: Could not update book", http.StatusInternalServerError)
		log.Printf("Error: Count not update book: %v", w)
		return
	}

	row, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error determining rows affected", http.StatusInternalServerError)
		fmt.Printf("Error determining rows affected: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"message":      fmt.Sprintf("Book Id: %v - %v updated successfully", book.Id, book.Title),
		"rows_updated": row,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "No matching param found", http.StatusInternalServerError)
		log.Printf("No matching param found: %v", err)
		return
	}

	result, err := db.Exec(`DELETE FROM books WHERE id LIKE ?`, id)
	if err != nil {
		http.Error(w, "No rows with matching id found", http.StatusInternalServerError)
		log.Printf("No rows with matching if found: %v", err)
		return
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error retrieving affected rows", http.StatusInternalServerError)
		log.Printf("Error retrieving affected rows: %v", err)
		return
	}

	log.Printf(" %v Book deleted successfully", rowsDeleted)
}
