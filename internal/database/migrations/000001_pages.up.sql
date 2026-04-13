CREATE TABLE pages (
                       id         TEXT PRIMARY KEY,
                       title      TEXT NOT NULL DEFAULT '',
                        icon      TEXT NOT NULL DEFAULT '',
                       parent_id UUID REFERENCES pages(id),
                       content    JSONB NOT NULL DEFAULT '[]',
                       version    INT NOT NULL DEFAULT 1,
                       updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

