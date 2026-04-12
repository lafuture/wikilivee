package database

import (
	"context"
	"encoding/json"
	"time"
	"wikilivee/internal/models"
)

func (p *Postgres) CreatePage(ctx context.Context, id, title string) (models.Page, error) {
	const q = `
		INSERT INTO pages (id, title, content, version, updated_at)
		VALUES ($1, $2, '[]', 1, NOW())
		RETURNING id, title, content, version, updated_at`

	return scanPage(p.Pool.QueryRow(ctx, q, id, title))
}

func (p *Postgres) GetPage(ctx context.Context, id string) (models.Page, error) {
	const q = `SELECT id, title, content, version, updated_at FROM pages WHERE id = $1`
	return scanPage(p.Pool.QueryRow(ctx, q, id))
}

func (p *Postgres) GetPages(ctx context.Context) ([]models.PageSummary, error) {
	const q = `SELECT id, title, updated_at FROM pages ORDER BY updated_at DESC`
	rows, err := p.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.PageSummary
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, s)
	}
	return pages, rows.Err()
}

func (p *Postgres) SavePage(ctx context.Context, id, title string, content []models.Block, version int) (int, error) {
	raw, err := json.Marshal(content)
	if err != nil {
		return 0, err
	}

	const q = `
		UPDATE pages
		SET title = $2, content = $3, version = version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING version`

	var newVersion int
	err = p.Pool.QueryRow(ctx, q, id, title, raw).Scan(&newVersion)
	return newVersion, err
}

func (p *Postgres) DeletePage(ctx context.Context, id string) error {
	const q = `DELETE FROM pages WHERE id = $1`
	_, err := p.Pool.Exec(ctx, q, id)
	return err
}

func (p *Postgres) GetPageBacklinks(ctx context.Context, id string) ([]models.PageSummary, error) {
	const q = `
		SELECT id, title, updated_at FROM pages
		WHERE content @> jsonb_build_array(
			jsonb_build_object(
				'type', 'page_link',
				'props', jsonb_build_object('targetId', $1)
			)
		)
		ORDER BY updated_at DESC`

	rows, err := p.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.PageSummary
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, s)
	}
	return pages, rows.Err()
}

type rower interface {
	Scan(dest ...any) error
}

func scanPage(row rower) (models.Page, error) {
	var p models.Page
	var contentRaw []byte
	var updatedAt time.Time

	if err := row.Scan(&p.ID, &p.Title, &contentRaw, &p.Version, &updatedAt); err != nil {
		return models.Page{}, err
	}
	p.UpdatedAt = updatedAt
	if err := json.Unmarshal(contentRaw, &p.Content); err != nil {
		return models.Page{}, err
	}
	return p, nil
}
