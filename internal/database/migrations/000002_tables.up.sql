CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tables (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE table_columns (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id   UUID NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT 'text',
    position   INT NOT NULL DEFAULT 0
);

CREATE TABLE table_rows (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_id   UUID NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE table_cells (
    row_id    UUID NOT NULL REFERENCES table_rows(id) ON DELETE CASCADE,
    column_id UUID NOT NULL REFERENCES table_columns(id) ON DELETE CASCADE,
    value     TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (row_id, column_id)
);
