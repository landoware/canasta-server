-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS games (
    room_code TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    game_data TEXT NOT NULL,  -- JSON serialized ActiveGame
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_updated_at ON games(updated_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    room_code TEXT NOT NULL,
    player_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (room_code) REFERENCES games(room_code) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_room_code ON sessions(room_code);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS room_codes (
    code TEXT PRIMARY KEY,
    in_use BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_room_codes_in_use ON room_codes(in_use);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS room_codes;
-- +goose StatementEnd
