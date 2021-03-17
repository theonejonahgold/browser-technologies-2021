package models

type Session struct {
	ID        string     `json:"_id"`
	Name      string     `json:"name"`
	Questions []Question `json:"questions"`
}
