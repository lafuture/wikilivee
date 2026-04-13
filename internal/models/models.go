package models

import "time"

type Block struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Content *string        `json:"content"`
	Props   map[string]any `json:"props"`
}

type Page struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Icon      string    `json:"icon"`
	ParentID  string    `json:"parent_id"`
	Content   []Block   `json:"content"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PageSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Icon      string    `json:"icon"`
	ParentID  string    `json:"parent_id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}
