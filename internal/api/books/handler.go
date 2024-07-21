package books

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	db   *sql.DB
	tmpl *template.Template
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:   db,
		tmpl: template.Must(template.ParseGlob("internal/templates/*.html")),
	}
}

func (h *Handler) GetBooks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`SELECT * FROM books`)
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

	if err := h.tmpl.ExecuteTemplate(w, "book-list", data); err != nil {
		log.Fatal(err)
	}
}

func (h *Handler) GetBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "No matching param in url found", http.StatusBadRequest)
		fmt.Printf("No matching param in url found: %v", err)
		return
	}

	result := h.db.QueryRow("SELECT * FROM books WHERE Id LIKE ?", id)
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

func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	isbn := r.FormValue("isbn")
	title := r.FormValue("title")
	author := r.FormValue("author")
	publisher := r.FormValue("publisher")
	createdAt := time.Now()

	result, err := h.db.Exec(`INSERT INTO books (isbn, title, author, publisher, created_at) VALUES (?,?,?,?,?)`,
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

	h.GetBooks(w, r)

	log.Printf("Book: %v, successfully created", bookId)
}

func (h *Handler) UpdateBook(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.db.Exec(`UPDATE books SET publisher = ? WHERE id LIKE ?`, book.Publisher, id)
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

func (h *Handler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "No matching param found", http.StatusInternalServerError)
		log.Printf("No matching param found: %v", err)
		return
	}

	result, err := h.db.Exec(`DELETE FROM books WHERE id LIKE ?`, id)
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

	h.GetBooks(w, r)
	log.Printf(" %v Book deleted successfully", rowsDeleted)
}
