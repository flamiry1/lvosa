-- +goose Up
CREATE TABLE exercises (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lesson_id  BIGINT NOT NULL,
    position   BIGINT NOT NULL CHECK (position > 0),
    type       TEXT NOT NULL CHECK (type IN ('choice', 'gap', 'inflect', 'match', 'order', 'true_false')),
    content    JSONB NOT NULL CHECK (jsonb_typeof(content) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT exercises_lesson_id_fkey FOREIGN KEY (lesson_id) REFERENCES lessons (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX exercises_position_idx ON exercises (lesson_id, position);

-- +goose Down
DROP TABLE exercises;
