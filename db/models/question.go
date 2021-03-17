package models

type Question struct {
	ID      string `json:"_id"`
	Title   string `json:"title"`
	Answers []struct {
		ID    string `json:"_id"`
		Title string `json:"title"`
	}
}
