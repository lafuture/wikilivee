CREATE TABLE page_versions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id    TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    version    INT NOT NULL,
    title      TEXT NOT NULL DEFAULT '',
    content    JSONB NOT NULL DEFAULT '[]',
    saved_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (page_id, version)
);
