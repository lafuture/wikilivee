CREATE TABLE pages (
                       id         TEXT PRIMARY KEY,
                       title      TEXT NOT NULL DEFAULT '',
                       content    JSONB NOT NULL DEFAULT '[]',
                       version    INT NOT NULL DEFAULT 1,
                       updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

