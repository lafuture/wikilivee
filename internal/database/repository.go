package database

import (
	"context"
	"encoding/json"
	"time"
	"wikilivee/internal/models"
)

func (p *Postgres) CreatePage(ctx context.Context, id, title, icon, parentId, author string) (models.Page, error) {
	const q = `
		INSERT INTO pages (id, title, icon, parent_id, author, content, version, updated_at)
		VALUES ($1, $2, $3, $4, $5, '[]', 1, NOW())
		RETURNING id, title, icon, COALESCE(parent_id, ''), content, version, updated_at`

	return scanPage(p.Pool.QueryRow(ctx, q, id, title, icon, parentId, author))
}

func (p *Postgres) GetPage(ctx context.Context, id string) (models.Page, error) {
	const q = `SELECT id, title, icon, COALESCE(parent_id, ''), content, version, updated_at FROM pages WHERE id = $1`
	return scanPage(p.Pool.QueryRow(ctx, q, id))
}

func (p *Postgres) GetPages(ctx context.Context) ([]models.PageSummary, error) {
	const q = `SELECT id, title, icon, COALESCE(parent_id, ''), version, updated_at FROM pages ORDER BY updated_at DESC`
	rows, err := p.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.PageSummary
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Icon, &s.ParentID, &s.Version, &s.UpdatedAt); err != nil {
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
	if err = p.Pool.QueryRow(ctx, q, id, title, raw).Scan(&newVersion); err != nil {
		return 0, err
	}

	const vq = `
		INSERT INTO page_versions (page_id, version, title, content, saved_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (page_id, version) DO NOTHING`
	_, _ = p.Pool.Exec(ctx, vq, id, newVersion, title, raw)

	return newVersion, nil
}

func (p *Postgres) DeletePage(ctx context.Context, id string) error {
	const q = `DELETE FROM pages WHERE id = $1`
	_, err := p.Pool.Exec(ctx, q, id)
	return err
}

func (p *Postgres) GetPageBacklinks(ctx context.Context, id string) ([]models.PageSummary, error) {
	const q = `
		SELECT id, title, updated_at FROM pages
		WHERE EXISTS (
			SELECT 1 FROM jsonb_array_elements(content) AS block
			WHERE block->>'type' = 'page_link'
			AND block->'props'->>'targetId' = $1
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

	if err := row.Scan(&p.ID, &p.Title, &p.Icon, &p.ParentID, &contentRaw, &p.Version, &updatedAt); err != nil {
		return models.Page{}, err
	}
	p.UpdatedAt = updatedAt
	if err := json.Unmarshal(contentRaw, &p.Content); err != nil {
		return models.Page{}, err
	}
	return p, nil
}

func (p *Postgres) GetPageChildren(ctx context.Context, id string) ([]models.PageSummary, error) {
	const q = `SELECT id, title, icon, COALESCE(parent_id, ''), version, updated_at FROM pages WHERE parent_id = $1`

	rows, err := p.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.PageSummary
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Icon, &s.ParentID, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, s)
	}

	return pages, rows.Err()
}

func (p *Postgres) GetPageVersions(ctx context.Context, id string) ([]models.PageVersionSummary, error) {
	const q = `
		SELECT version, saved_at
		FROM page_versions
		WHERE page_id = $1
		ORDER BY version DESC`

	rows, err := p.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.PageVersionSummary
	for rows.Next() {
		var v models.PageVersionSummary
		if err := rows.Scan(&v.Version, &v.SavedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []models.PageVersionSummary{}
	}
	return versions, rows.Err()
}

func (p *Postgres) GetPageVersion(ctx context.Context, id string, version string) (models.PageVersion, error) {
	const q = `
		SELECT page_id, version, title, content, saved_at
		FROM page_versions
		WHERE page_id = $1 AND version = $2`

	var pv models.PageVersion
	var contentRaw []byte
	if err := p.Pool.QueryRow(ctx, q, id, version).Scan(
		&pv.PageID, &pv.Version, &pv.Title, &contentRaw, &pv.SavedAt,
	); err != nil {
		return models.PageVersion{}, err
	}
	if err := json.Unmarshal(contentRaw, &pv.Content); err != nil {
		return models.PageVersion{}, err
	}
	return pv, nil
}

func (p *Postgres) RestorePage(ctx context.Context, id string, version string) (int, error) {
	const q = `
		UPDATE pages
		SET title      = pv.title,
		    content    = pv.content,
		    version    = pages.version + 1,
		    updated_at = NOW()
		FROM page_versions pv
		WHERE pages.id = $1
		  AND pv.page_id = $1
		  AND pv.version = $2::int
		RETURNING pages.version, pv.title, pv.content`

	var newVersion int
	var title string
	var contentRaw []byte
	if err := p.Pool.QueryRow(ctx, q, id, version).Scan(&newVersion, &title, &contentRaw); err != nil {
		return 0, err
	}

	const vq = `
		INSERT INTO page_versions (page_id, version, title, content, saved_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (page_id, version) DO NOTHING`
	_, _ = p.Pool.Exec(ctx, vq, id, newVersion, title, contentRaw)

	return newVersion, nil
}

// ---- User repository methods ----

func (p *Postgres) CreateUser(ctx context.Context, username, passwordHash string) (models.User, error) {
	const q = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, username, password_hash, created_at`

	var u models.User
	err := p.Pool.QueryRow(ctx, q, username, passwordHash).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

func (p *Postgres) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	const q = `SELECT id::text, username, password_hash, created_at FROM users WHERE username = $1`
	var u models.User
	err := p.Pool.QueryRow(ctx, q, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

// ---- Comment repository methods ----

func (p *Postgres) GetComments(ctx context.Context, pageID string) ([]models.Comment, error) {
	const q = `
		SELECT id::text, page_id, author, text, anchor_from, anchor_to, anchor_text, created_at
		FROM comments
		WHERE page_id = $1
		ORDER BY created_at ASC`

	rows, err := p.Pool.Query(ctx, q, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.PageID, &c.Author, &c.Text, &c.AnchorFrom, &c.AnchorTo, &c.AnchorText, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	if comments == nil {
		comments = []models.Comment{}
	}
	return comments, rows.Err()
}

func (p *Postgres) CreateComment(ctx context.Context, pageID, author, text string, anchorFrom, anchorTo int, anchorText string) (models.Comment, error) {
	const q = `
		INSERT INTO comments (page_id, author, text, anchor_from, anchor_to, anchor_text)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, page_id, author, text, anchor_from, anchor_to, anchor_text, created_at`

	var c models.Comment
	err := p.Pool.QueryRow(ctx, q, pageID, author, text, anchorFrom, anchorTo, anchorText).
		Scan(&c.ID, &c.PageID, &c.Author, &c.Text, &c.AnchorFrom, &c.AnchorTo, &c.AnchorText, &c.CreatedAt)
	return c, err
}

func (p *Postgres) DeleteComment(ctx context.Context, pageID, commentID string) error {
	const q = `DELETE FROM comments WHERE id = $1::uuid AND page_id = $2`
	_, err := p.Pool.Exec(ctx, q, commentID, pageID)
	return err
}

// ---- Table repository methods ----

func (p *Postgres) GetTables(ctx context.Context) ([]models.TableSummary, error) {
	const q = `
		SELECT id::text, name, created_at, updated_at
		FROM tables
		ORDER BY created_at DESC`

	rows, err := p.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.TableSummary
	for rows.Next() {
		var s models.TableSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		tables = append(tables, s)
	}
	return tables, rows.Err()
}

func (p *Postgres) GetTable(ctx context.Context, id string) (*models.Table, error) {
	const tq = `
		SELECT id::text, name, created_at, updated_at
		FROM tables WHERE id = $1::uuid`

	var t models.Table
	if err := p.Pool.QueryRow(ctx, tq, id).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}

	const cq = `
		SELECT id::text, table_id::text, name, type, position
		FROM table_columns
		WHERE table_id = $1::uuid
		ORDER BY position`

	colRows, err := p.Pool.Query(ctx, cq, id)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	t.Columns = []models.TableColumn{}
	for colRows.Next() {
		var c models.TableColumn
		if err := colRows.Scan(&c.ID, &c.TableID, &c.Name, &c.Type, &c.Position); err != nil {
			return nil, err
		}
		t.Columns = append(t.Columns, c)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	const rq = `
		SELECT r.id::text, r.table_id::text, r.created_at,
		       COALESCE(c.id::text, '') AS col_id, COALESCE(ce.value, '') AS cell_val
		FROM table_rows r
		LEFT JOIN table_columns c ON c.table_id = r.table_id
		LEFT JOIN table_cells ce ON ce.row_id = r.id AND ce.column_id = c.id
		WHERE r.table_id = $1::uuid
		ORDER BY r.created_at, c.position`

	cellRows, err := p.Pool.Query(ctx, rq, id)
	if err != nil {
		return nil, err
	}
	defer cellRows.Close()

	rowMap := make(map[string]*models.TableRow)
	var rowOrder []string

	for cellRows.Next() {
		var rowID, tableID, colID, cellVal string
		var createdAt time.Time
		if err := cellRows.Scan(&rowID, &tableID, &createdAt, &colID, &cellVal); err != nil {
			return nil, err
		}
		if _, ok := rowMap[rowID]; !ok {
			rowMap[rowID] = &models.TableRow{
				ID:        rowID,
				TableID:   tableID,
				CreatedAt: createdAt,
				Values:    make(map[string]string),
			}
			rowOrder = append(rowOrder, rowID)
		}
		if colID != "" {
			rowMap[rowID].Values[colID] = cellVal
		}
	}
	if err := cellRows.Err(); err != nil {
		return nil, err
	}

	t.Rows = make([]models.TableRow, 0, len(rowOrder))
	for _, rid := range rowOrder {
		t.Rows = append(t.Rows, *rowMap[rid])
	}

	return &t, nil
}

func (p *Postgres) CreateTable(ctx context.Context, name string, columns []models.TableColumnSpec) (*models.Table, error) {
	const tq = `
		INSERT INTO tables (name) VALUES ($1)
		RETURNING id::text, name, created_at, updated_at`

	var t models.Table
	if err := p.Pool.QueryRow(ctx, tq, name).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}

	t.Columns = []models.TableColumn{}
	t.Rows = []models.TableRow{}

	for i, col := range columns {
		const cq = `
			INSERT INTO table_columns (table_id, name, type, position)
			VALUES ($1::uuid, $2, $3, $4)
			RETURNING id::text, table_id::text, name, type, position`

		var c models.TableColumn
		if err := p.Pool.QueryRow(ctx, cq, t.ID, col.Name, col.Type, i).Scan(
			&c.ID, &c.TableID, &c.Name, &c.Type, &c.Position,
		); err != nil {
			return nil, err
		}
		t.Columns = append(t.Columns, c)
	}

	return &t, nil
}

func (p *Postgres) UpdateTable(ctx context.Context, id, name string) (*models.Table, error) {
	const q = `
		UPDATE tables SET name = $2, updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING id::text, name, created_at, updated_at`

	var t models.Table
	if err := p.Pool.QueryRow(ctx, q, id, name).Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &t, nil
}

func (p *Postgres) DeleteTable(ctx context.Context, id string) error {
	const q = `DELETE FROM tables WHERE id = $1::uuid`
	_, err := p.Pool.Exec(ctx, q, id)
	return err
}

func (p *Postgres) AddColumn(ctx context.Context, tableID, name, colType string) (*models.TableColumn, error) {
	const posQ = `
		SELECT COALESCE(MAX(position) + 1, 0)
		FROM table_columns WHERE table_id = $1::uuid`
	var pos int
	if err := p.Pool.QueryRow(ctx, posQ, tableID).Scan(&pos); err != nil {
		return nil, err
	}

	const q = `
		INSERT INTO table_columns (table_id, name, type, position)
		VALUES ($1::uuid, $2, $3, $4)
		RETURNING id::text, table_id::text, name, type, position`

	var c models.TableColumn
	if err := p.Pool.QueryRow(ctx, q, tableID, name, colType, pos).Scan(
		&c.ID, &c.TableID, &c.Name, &c.Type, &c.Position,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

func (p *Postgres) DeleteColumn(ctx context.Context, tableID, columnID string) error {
	const q = `
		DELETE FROM table_columns
		WHERE id = $1::uuid AND table_id = $2::uuid`
	_, err := p.Pool.Exec(ctx, q, columnID, tableID)
	return err
}

func (p *Postgres) AddRow(ctx context.Context, tableID string) (*models.TableRow, error) {
	const q = `
		INSERT INTO table_rows (table_id) VALUES ($1::uuid)
		RETURNING id::text, table_id::text, created_at`

	var row models.TableRow
	if err := p.Pool.QueryRow(ctx, q, tableID).Scan(
		&row.ID, &row.TableID, &row.CreatedAt,
	); err != nil {
		return nil, err
	}
	row.Values = map[string]string{}
	return &row, nil
}

func (p *Postgres) UpdateRow(ctx context.Context, tableID, rowID string, values map[string]string) (*models.TableRow, error) {
	const checkQ = `SELECT id::text FROM table_rows WHERE id = $1::uuid AND table_id = $2::uuid`
	var checkID string
	if err := p.Pool.QueryRow(ctx, checkQ, rowID, tableID).Scan(&checkID); err != nil {
		return nil, err
	}

	const upsertQ = `
		INSERT INTO table_cells (row_id, column_id, value)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (row_id, column_id) DO UPDATE SET value = EXCLUDED.value`

	for colID, val := range values {
		if _, err := p.Pool.Exec(ctx, upsertQ, rowID, colID, val); err != nil {
			return nil, err
		}
	}

	const rq = `
		SELECT c.id::text AS col_id, COALESCE(ce.value, '') AS cell_val
		FROM table_columns c
		LEFT JOIN table_cells ce ON ce.row_id = $1::uuid AND ce.column_id = c.id
		WHERE c.table_id = $2::uuid
		ORDER BY c.position`

	cellRows, err := p.Pool.Query(ctx, rq, rowID, tableID)
	if err != nil {
		return nil, err
	}
	defer cellRows.Close()

	row := &models.TableRow{
		ID:      rowID,
		TableID: tableID,
		Values:  make(map[string]string),
	}
	for cellRows.Next() {
		var colID, cellVal string
		if err := cellRows.Scan(&colID, &cellVal); err != nil {
			return nil, err
		}
		row.Values[colID] = cellVal
	}
	if err := cellRows.Err(); err != nil {
		return nil, err
	}

	return row, nil
}

func (p *Postgres) DeleteRow(ctx context.Context, tableID, rowID string) error {
	const q = `
		DELETE FROM table_rows
		WHERE id = $1::uuid AND table_id = $2::uuid`
	_, err := p.Pool.Exec(ctx, q, rowID, tableID)
	return err
}
