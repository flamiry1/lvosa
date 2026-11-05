-- name: InsertLesson :one
INSERT INTO lessons (owner_id, title, explanation, level, topic, difficulty, focus,  size, visibility, copied_from_id, moderation_status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id;

-- name: InsertExercise :one
INSERT INTO exercises (lesson_id, position, type, content) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetLessonByID :one
SELECT * FROM lessons WHERE id = $1;

-- name: LessonExercisesByOrderAscPosition :many
SELECT * FROM exercises WHERE lesson_id = $1 ORDER BY position ASC;

-- name: OwnerLessons :many
SELECT * FROM lessons WHERE owner_id = $1 ORDER BY id DESC;

-- name: MarketLessonsPublicApproved :many
SELECT * FROM lessons WHERE moderation_status = 'approved' AND visibility = 'public' ORDER BY saves_count DESC, id DESC LIMIT $1;

-- name: LessonPublicApproved :one
SELECT * FROM lessons WHERE moderation_status = 'approved' AND id = $1;

-- name: IncreaseSavesCount :exec
UPDATE lessons SET saves_count = (saves_count + 1) WHERE id = $1;
