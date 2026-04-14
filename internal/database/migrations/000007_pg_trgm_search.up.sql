CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_pages_title_trgm
ON pages USING gin (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_pages_content_trgm
ON pages USING gin ((content::text) gin_trgm_ops);
