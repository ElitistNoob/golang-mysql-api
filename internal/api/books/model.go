package books

import "time"

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
