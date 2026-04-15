package database

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
	"wikilivee/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *Postgres) CreatePage(ctx context.Context, id, title, icon, parentId, author string) (models.Page, error) {
	const q = `
		INSERT INTO pages (id, title, icon, author, parent_id, content, version, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), '[]', 1, NOW())
		RETURNING id, title, icon, author, COALESCE(parent_id, ''), content, version, updated_at`

	return scanPage(p.Pool.QueryRow(ctx, q, id, title, icon, author, parentId))
}

func (p *Postgres) GetPage(ctx context.Context, id string) (models.Page, error) {
	const q = `SELECT id, title, icon, author, COALESCE(parent_id, ''), content, version, updated_at FROM pages WHERE id = $1`
	return scanPage(p.Pool.QueryRow(ctx, q, id))
}

func (p *Postgres) GetPages(ctx context.Context) ([]models.PageSummary, error) {
	const q = `SELECT id, title, icon, author, COALESCE(parent_id, ''), version, updated_at FROM pages ORDER BY updated_at DESC`
	rows, err := p.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []models.PageSummary
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Icon, &s.Author, &s.ParentID, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, s)
	}
	return pages, rows.Err()
}

func (p *Postgres) GetPagesForUser(ctx context.Context, userID, username string) ([]models.PageSummary, error) {
	const q = `
		SELECT DISTINCT p.id, p.title, p.icon, p.author, COALESCE(p.parent_id, ''), p.version, p.updated_at
		FROM pages p
		LEFT JOIN page_permissions pp
		  ON pp.page_id = p.id
		 AND pp.user_id = $1::uuid
		 AND pp.role = 'editor'
		WHERE p.author = $2
		   OR pp.user_id IS NOT NULL
		ORDER BY p.updated_at DESC`
	rows, err := p.Pool.Query(ctx, q, userID, username)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return p.getPagesByAuthor(ctx, username)
		}
		return nil, err
	}
	defer rows.Close()

	items := make([]models.PageSummary, 0)
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Icon, &s.Author, &s.ParentID, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (p *Postgres) getPagesByAuthor(ctx context.Context, username string) ([]models.PageSummary, error) {
	const q = `
		SELECT id, title, icon, author, COALESCE(parent_id, ''), version, updated_at
		FROM pages
		WHERE author = $1
		ORDER BY updated_at DESC`
	rows, err := p.Pool.Query(ctx, q, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.PageSummary, 0)
	for rows.Next() {
		var s models.PageSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Icon, &s.Author, &s.ParentID, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (p *Postgres) SearchPages(ctx context.Context, text string) ([]models.SearchResult, error) {
	query := strings.TrimSpace(text)
	if query == "" {
		return []models.SearchResult{}, nil
	}

	const shortQuerySQL = `
		WITH searchable AS (
			SELECT
				p.id,
				p.title,
				p.updated_at,
				COALESCE((
					SELECT string_agg(trim(block->>'content'), ' ')
					FROM jsonb_array_elements(
						CASE WHEN jsonb_typeof(p.content) = 'array' THEN p.content ELSE '[]'::jsonb END
					) AS block
					WHERE block ? 'content'
					  AND trim(block->>'content') <> ''
				), '') AS body
			FROM pages p
		)
		SELECT
			id,
			title,
			CASE
				WHEN position(lower($1) in lower(body)) > 0 THEN substr(body, greatest(position(lower($1) in lower(body)) - 40, 1), 160)
				WHEN body <> '' THEN left(body, 160)
				ELSE left(title, 160)
			END AS snippet,
			updated_at
		FROM searchable
		WHERE lower(title) LIKE lower('%' || $1 || '%')
		   OR lower(body) LIKE lower('%' || $1 || '%')
		ORDER BY updated_at DESC
		LIMIT 20`

	if utf8.RuneCountInString(query) < 3 {
		rows, err := p.Pool.Query(ctx, shortQuerySQL, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		results := make([]models.SearchResult, 0)
		for rows.Next() {
			var item models.SearchResult
			if err := rows.Scan(&item.PageID, &item.Title, &item.Snippet, &item.UpdatedAt); err != nil {
				return nil, err
			}
			results = append(results, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return results, nil
	}

	const trigramQuerySQL = `
		WITH searchable AS (
			SELECT
				p.id,
				p.title,
				p.updated_at,
				COALESCE((
					SELECT string_agg(trim(block->>'content'), ' ')
					FROM jsonb_array_elements(
						CASE WHEN jsonb_typeof(p.content) = 'array' THEN p.content ELSE '[]'::jsonb END
					) AS block
					WHERE block ? 'content'
					  AND trim(block->>'content') <> ''
				), '') AS body
			FROM pages p
		),
		ranked AS (
			SELECT
				id,
				title,
				updated_at,
				body,
				GREATEST(
					similarity(title, $1),
					similarity(body, $1),
					similarity(title || ' ' || body, $1)
				) AS score
			FROM searchable
			WHERE title % $1
			   OR body % $1
			   OR (title || ' ' || body) % $1
			   OR lower(title) LIKE lower('%' || $1 || '%')
			   OR lower(body) LIKE lower('%' || $1 || '%')
		)
		SELECT
			id,
			title,
			CASE
				WHEN position(lower($1) in lower(body)) > 0 THEN substr(body, greatest(position(lower($1) in lower(body)) - 40, 1), 160)
				WHEN body <> '' THEN left(body, 160)
				ELSE left(title, 160)
			END AS snippet,
			updated_at
		FROM ranked
		ORDER BY score DESC, updated_at DESC
		LIMIT 20`

	rows, err := p.Pool.Query(ctx, trigramQuerySQL, query)
	if err != nil {
		rows, err = p.Pool.Query(ctx, shortQuerySQL, query)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	results := make([]models.SearchResult, 0)
	for rows.Next() {
		var item models.SearchResult
		if err := rows.Scan(&item.PageID, &item.Title, &item.Snippet, &item.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
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
		SELECT id, title, icon, COALESCE(parent_id, ''), version, updated_at FROM pages
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
		if err := rows.Scan(&s.ID, &s.Title, &s.Icon, &s.ParentID, &s.Version, &s.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, s)
	}
	return pages, rows.Err()
}

func (p *Postgres) GetPagesGraph(ctx context.Context) (models.Graph, error) {

	nodes := make([]models.GraphNode, 0)
	edges := make([]models.GraphEdge, 0)

	const nodesQ = `
		SELECT id, COALESCE(NULLIF(title, ''), 'Untitled') AS title
		FROM pages
		ORDER BY updated_at DESC`

	nodeRows, err := p.Pool.Query(ctx, nodesQ)
	if err != nil {
		return models.Graph{Nodes: nodes, Edges: edges}, err
	}
	defer nodeRows.Close()

	validIDs := make(map[string]struct{})
	for nodeRows.Next() {
		var node models.GraphNode
		if err := nodeRows.Scan(&node.ID, &node.Title); err != nil {
			return models.Graph{Nodes: nodes, Edges: edges}, err
		}
		nodes = append(nodes, node)
		validIDs[node.ID] = struct{}{}
	}
	if err := nodeRows.Err(); err != nil {
		return models.Graph{Nodes: nodes, Edges: edges}, err
	}

	if len(nodes) == 0 {
		return models.Graph{Nodes: nodes, Edges: edges}, nil
	}

	const hierarchyQ = `
		SELECT id, parent_id
		FROM pages
		WHERE parent_id IS NOT NULL
		  AND parent_id <> ''`

	hierarchyRows, err := p.Pool.Query(ctx, hierarchyQ)
	if err == nil {
		defer hierarchyRows.Close()
		seen := make(map[string]struct{})
		for hierarchyRows.Next() {
			var child string
			var parent *string
			if err := hierarchyRows.Scan(&child, &parent); err != nil {
				continue
			}
			if parent == nil || *parent == "" || child == "" {
				continue
			}
			if _, ok := validIDs[*parent]; !ok {
				continue
			}
			if _, ok := validIDs[child]; !ok {
				continue
			}
			key := child + "->" + *parent
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			edges = append(edges, models.GraphEdge{
				Source: child,
				Target: *parent,
			})
		}
	}

	const edgesQ = `
		WITH block_refs AS (
			SELECT
				p.id AS source,
				block
			FROM pages p
			CROSS JOIN LATERAL jsonb_array_elements(
				CASE WHEN jsonb_typeof(p.content) = 'array' THEN p.content ELSE '[]'::jsonb END
			) AS block
		),
		targets AS (
			SELECT
				source,
				NULLIF(
					COALESCE(block->'props'->>'targetId', block->'props'->>'pageId'),
					''
				) AS target
			FROM block_refs
			WHERE block->>'type' = 'page_link'

			UNION ALL

			SELECT
				source,
				NULLIF(link #>> '{}', '') AS target
			FROM block_refs
			CROSS JOIN LATERAL jsonb_path_query(
				block,
				'$.props.tiptapContent.** ? (@.type == "wikiLink").attrs.pageId'
			) AS link
		)
		SELECT source, target
		FROM targets
		WHERE target IS NOT NULL`

	edgeRows, err := p.Pool.Query(ctx, edgesQ)
	if err != nil {
		return models.Graph{Nodes: nodes, Edges: edges}, nil
	}
	defer edgeRows.Close()

	seen := make(map[string]struct{}, len(edges))
	for _, edge := range edges {
		seen[edge.Source+"->"+edge.Target] = struct{}{}
	}
	for edgeRows.Next() {
		var source string
		var target *string
		if err := edgeRows.Scan(&source, &target); err != nil {
			continue
		}
		if target == nil || *target == "" || source == "" {
			continue
		}
		if _, ok := validIDs[*target]; !ok {
			continue
		}
		key := source + "->" + *target
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		edges = append(edges, models.GraphEdge{
			Source: source,
			Target: *target,
		})
	}

	return models.Graph{Nodes: nodes, Edges: edges}, nil
}

type rower interface {
	Scan(dest ...any) error
}

func scanPage(row rower) (models.Page, error) {
	var p models.Page
	var contentRaw []byte
	var updatedAt time.Time

	if err := row.Scan(&p.ID, &p.Title, &p.Icon, &p.Author, &p.ParentID, &contentRaw, &p.Version, &updatedAt); err != nil {
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

func (p *Postgres) SearchUsers(ctx context.Context, query string, limit int) ([]models.UserSummary, error) {
	normalized := strings.TrimSpace(query)
	if normalized == "" {
		return []models.UserSummary{}, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	const q = `
		SELECT id::text, username
		FROM users
		WHERE username ILIKE '%' || $1 || '%'
		ORDER BY username
		LIMIT $2`

	rows, err := p.Pool.Query(ctx, q, normalized, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.UserSummary, 0)
	for rows.Next() {
		var item models.UserSummary
		if err := rows.Scan(&item.ID, &item.Username); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []models.UserSummary{}
	}
	return items, rows.Err()
}

func (p *Postgres) IsPageOwner(ctx context.Context, pageID, username string) (bool, error) {
	const q = `SELECT 1 FROM pages WHERE id = $1 AND author = $2`
	var marker int
	if err := p.Pool.QueryRow(ctx, q, pageID, username).Scan(&marker); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return marker == 1, nil
}

func (p *Postgres) CanUserEditPage(ctx context.Context, pageID, userID, username string) (bool, error) {
	owner, err := p.IsPageOwner(ctx, pageID, username)
	if err != nil {
		return false, err
	}
	if owner {
		return true, nil
	}

	const q = `
		SELECT 1
		FROM page_permissions
		WHERE page_id = $1
		  AND user_id = $2::uuid
		  AND role = 'editor'`
	var marker int
	if err := p.Pool.QueryRow(ctx, q, pageID, userID).Scan(&marker); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return false, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return marker == 1, nil
}

func (p *Postgres) GetPageAccessEntries(ctx context.Context, pageID string) ([]models.PageAccessEntry, error) {
	const q = `
		SELECT u.id::text, u.username, pp.role
		FROM page_permissions pp
		JOIN users u ON u.id = pp.user_id
		WHERE pp.page_id = $1
		ORDER BY pp.created_at ASC`

	rows, err := p.Pool.Query(ctx, q, pageID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return []models.PageAccessEntry{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	items := make([]models.PageAccessEntry, 0)
	for rows.Next() {
		var item models.PageAccessEntry
		if err := rows.Scan(&item.UserID, &item.Username, &item.Role); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if items == nil {
		items = []models.PageAccessEntry{}
	}
	return items, rows.Err()
}

func (p *Postgres) UpsertPagePermission(ctx context.Context, pageID, userID, role string) error {
	const q = `
		INSERT INTO page_permissions (page_id, user_id, role)
		VALUES ($1, $2::uuid, $3)
		ON CONFLICT (page_id, user_id)
		DO UPDATE SET role = EXCLUDED.role`
	_, err := p.Pool.Exec(ctx, q, pageID, userID, role)
	return err
}

func (p *Postgres) DeletePagePermission(ctx context.Context, pageID, userID string) error {
	const q = `DELETE FROM page_permissions WHERE page_id = $1 AND user_id = $2::uuid`
	_, err := p.Pool.Exec(ctx, q, pageID, userID)
	return err
}

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

func (p *Postgres) DeleteComment(ctx context.Context, pageID, commentID, author string) error {
	const q = `DELETE FROM comments WHERE id = $1::uuid AND page_id = $2 AND author = $3`
	result, err := p.Pool.Exec(ctx, q, commentID, pageID, author)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

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
