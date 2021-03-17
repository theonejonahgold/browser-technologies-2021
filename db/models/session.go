package models

type Session struct {
	ID        string     `json:"_id"`
	Name      string     `json:"name"`
	Owner     string     `json:"owner"`
	Questions []Question `json:"questions"`
}
