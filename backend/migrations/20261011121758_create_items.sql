-- +goose Up
CREATE TABLE items
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX items_name_lower_idx on items (lower(name));

-- +goose Down
DROP TABLE items;
