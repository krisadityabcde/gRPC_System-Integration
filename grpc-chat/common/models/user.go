package models

type User struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}
