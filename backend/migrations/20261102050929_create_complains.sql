-- +goose Up
CREATE TABLE complains (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lesson_id  BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    reason     TEXT NOT NULL CHECK (reason IN ('wrong_content', 'offensive', 'spam', 'other')),
    comment    TEXT CHECK (char_length(comment) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT complains_lesson_id_fkey FOREIGN KEY (lesson_id) REFERENCES lessons (id) ON DELETE CASCADE,
    CONSTRAINT complains_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX complains_user_id_idx ON complains (user_id);
CREATE UNIQUE INDEX complains_lesson_id_user_id_idx ON complains (lesson_id, user_id);

-- +goose Down
DROP TABLE complains;
