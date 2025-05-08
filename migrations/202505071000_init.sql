-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS clients (
    api_key TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    capacity INT NOT NULL,
    rate INT NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS clients;

-- +goose StatementEnd