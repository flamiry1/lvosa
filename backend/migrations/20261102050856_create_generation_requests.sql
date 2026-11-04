-- +goose Up
CREATE TABLE generation_requests (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    lesson_id   BIGINT,
    level       TEXT NOT NULL CHECK (level IN ('A1', 'A2', 'B1', 'B2', 'C1')),
    topic       TEXT NOT NULL CHECK (char_length(topic) <= 80),
    difficulty  TEXT NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    focus       TEXT NOT NULL CHECK (focus IN ('grammar', 'vocabulary', 'reading', 'mixed')),
    size        TEXT NOT NULL CHECK (size IN ('short', 'normal', 'long')),
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    error       TEXT,
    attempts    INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT generation_requests_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT generation_requests_lesson_id_fkey FOREIGN KEY (lesson_id) REFERENCES lessons (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX generation_requests_user_id_active_idx ON generation_requests (user_id)
WHERE
status = 'pending' OR status = 'processing';
CREATE INDEX generation_requests_created_at_idx ON generation_requests (created_at)
WHERE
status = 'pending';
CREATE INDEX generation_requests_lesson_id_idx ON generation_requests (lesson_id);
CREATE INDEX generation_requests_user_idx ON generation_requests (user_id);

-- +goose Down
DROP TABLE generation_requests;
