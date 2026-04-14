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

type PageVersionSummary struct {
	Version int       `json:"version"`
	SavedAt time.Time `json:"savedAt"`
}

type PageVersion struct {
	PageID  string    `json:"pageId"`
	Version int       `json:"version"`
	Title   string    `json:"title"`
	Content []Block   `json:"content"`
	SavedAt time.Time `json:"savedAt"`
}

type TableColumnSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type TableColumn struct {
	ID       string `json:"id"`
	TableID  string `json:"table_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Position int    `json:"position"`
}

type TableRow struct {
	ID        string            `json:"id"`
	TableID   string            `json:"table_id"`
	CreatedAt time.Time         `json:"created_at"`
	Values    map[string]string `json:"values"`
}

type Table struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Columns   []TableColumn `json:"columns"`
	Rows      []TableRow    `json:"rows"`
}

type TableSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Comment struct {
	ID         string    `json:"id"`
	PageID     string    `json:"pageId"`
	Author     string    `json:"author"`
	Text       string    `json:"text"`
	AnchorFrom int       `json:"anchorFrom"`
	AnchorTo   int       `json:"anchorTo"`
	AnchorText string    `json:"anchorText"`
	CreatedAt  time.Time `json:"createdAt"`
}
