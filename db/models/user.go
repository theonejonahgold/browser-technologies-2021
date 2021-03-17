package models

type User struct {
	ID       string `json:"_id"`
	Username string `json:"username"`
	Password string `json:"-"`
}
