CREATE TABLE page_permissions (
    page_id    TEXT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'editor',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (page_id, user_id),
    CONSTRAINT page_permissions_role_check CHECK (role IN ('editor'))
);

CREATE INDEX idx_page_permissions_page ON page_permissions(page_id);
CREATE INDEX idx_page_permissions_user ON page_permissions(user_id);
