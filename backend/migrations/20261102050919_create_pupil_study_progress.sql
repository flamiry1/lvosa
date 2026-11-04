-- +goose Up
CREATE TABLE pupil_study_progress (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    exercise_id BIGINT NOT NULL,
    is_correct  BOOLEAN NOT NULL,
    attempts    INT NOT NULL CHECK (attempts >= 1),
    answered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pupil_study_progress_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT pupil_study_progress_exercise_id_fkey FOREIGN KEY (exercise_id) REFERENCES exercises (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX pupil_study_progress_user_id_exercise_id_idx ON pupil_study_progress (user_id, exercise_id);
CREATE INDEX pupil_study_progress_exercise_id_idx ON pupil_study_progress (exercise_id);

-- +goose Down
DROP TABLE pupil_study_progress;
