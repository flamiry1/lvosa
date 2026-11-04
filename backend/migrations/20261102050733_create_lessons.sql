-- +goose Up
CREATE TABLE lessons (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id          BIGINT NOT NULL,
    title             TEXT NOT NULL CHECK (char_length(title) <= 120),
    explanation       TEXT NOT NULL CHECK (char_length(explanation) <= 5000),
    level             TEXT NOT NULL CHECK (level IN ('A1', 'A2', 'B1', 'B2', 'C1')),
    topic             TEXT NOT NULL CHECK (char_length(topic) <= 80),
    difficulty        TEXT NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    focus             TEXT NOT NULL CHECK (focus IN ('grammar', 'vocabulary', 'reading', 'mixed')),
    size              TEXT NOT NULL CHECK (size IN ('short', 'normal', 'long')),
    visibility        TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('public', 'private')),
    copied_from_id    BIGINT,
    moderation_status TEXT CHECK (moderation_status IN ('pending', 'approved', 'rejected', 'hidden')),
    saves_count       INT NOT NULL DEFAULT 0 CHECK (saves_count >= 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT lessons_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT lessons_copied_from_id_fkey FOREIGN KEY (copied_from_id) REFERENCES lessons (id) ON DELETE SET NULL,
    CONSTRAINT lessons_moderation_status_chk CHECK ((visibility = 'private' AND moderation_status IS NULL) OR (visibility = 'public' AND moderation_status IS NOT NULL))
);

CREATE INDEX lessons_owner_id_idx ON lessons (owner_id);

CREATE INDEX lessons_copied_from_id_idx ON lessons (copied_from_id);

CREATE INDEX lessons_marketplace_idx ON lessons (saves_count DESC, id DESC)
WHERE
visibility = 'public' AND moderation_status = 'approved';

-- +goose Down
DROP TABLE lessons;
