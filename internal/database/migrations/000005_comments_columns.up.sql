CREATE TABLE comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id     TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    author      TEXT NOT NULL DEFAULT '',
    text        TEXT NOT NULL DEFAULT '',
    anchor_from INT  NOT NULL DEFAULT 0,
    anchor_to   INT  NOT NULL DEFAULT 0,
    anchor_text TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
