package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/flamiry1/lvosa/backend/internal/models"
	"github.com/flamiry1/lvosa/backend/internal/store/postgres/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LessonRepo struct {
	pool *pgxpool.Pool
	db   *db.Queries
}

func NewLessonRepo(p *pgxpool.Pool) *LessonRepo {
	return &LessonRepo{pool: p, db: db.New(p)}
}

func int8Ptr(i pgtype.Int8) *int {
	if !i.Valid {
		return nil
	}
	val := int(i.Int64)
	return &val
}

func textPtrModerationStatus(i pgtype.Text) *models.ModerationStatus {
	if !i.Valid {
		return nil
	}
	val := models.ModerationStatus(i.String)
	return &val
}

func transferToModelsLesson(l db.Lesson) models.Lesson {
	return models.Lesson{
		ID:               int(l.ID),
		OwnerID:          int(l.OwnerID),
		Title:            l.Title,
		Explanation:      l.Explanation,
		Level:            models.Level(l.Level),
		Topic:            l.Topic,
		Difficulty:       models.Difficulty(l.Difficulty),
		Focus:            models.Focus(l.Focus),
		Size:             models.Size(l.Size),
		Visibility:       models.Visibility(l.Visibility),
		CopiedFromID:     int8Ptr(l.CopiedFromID),
		ModerationStatus: textPtrModerationStatus(l.ModerationStatus),
		SavesCount:       int(l.SavesCount),
		CreatedAt:        l.CreatedAt.Time,
	}
}

func fillLessonWithExercises(exs []db.Exercise, l *models.Lesson) {
	for _, e := range exs {
		exercisesModel := models.Exercise{
			ID:        int(e.ID),
			LessonID:  l.ID,
			Position:  int(e.Position),
			Type:      models.ExerciseType(e.Type),
			Content:   e.Content,
			CreatedAt: e.CreatedAt.Time,
		}
		l.Exercises = append(l.Exercises, exercisesModel)
	}
}

func (l *LessonRepo) CreateWithExercises(ctx context.Context, lesson models.Lesson, exercises []models.Exercise) (int, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create with exercises transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := l.db.WithTx(tx)

	params := db.InsertLessonParams{
		OwnerID:     int64(lesson.OwnerID),
		Title:       lesson.Title,
		Explanation: lesson.Explanation,
		Level:       string(lesson.Level),
		Topic:       lesson.Topic,
		Difficulty:  string(lesson.Difficulty),
		Focus:       string(lesson.Focus),
		Size:        string(lesson.Size),
		Visibility:  string(lesson.Visibility),
	}
	if lesson.CopiedFromID != nil {
		params.CopiedFromID = pgtype.Int8{Int64: int64(*lesson.CopiedFromID), Valid: true}
	}
	if lesson.ModerationStatus != nil {
		params.ModerationStatus = pgtype.Text{String: string(*lesson.ModerationStatus), Valid: true}
	}
	lessonID, err := q.InsertLesson(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("create lesson: insert lesson: %w", err)
	}
	for _, e := range exercises {
		_, err = q.InsertExercise(ctx, db.InsertExerciseParams{
			LessonID: lessonID,
			Position: int64(e.Position),
			Type:     string(e.Type),
			Content:  []byte(e.Content),
		})
		if err != nil {
			return 0, fmt.Errorf("create lesson: insert exerice %d: %w", e.Position, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("create lesson: commit: %w", err)
	}

	return int(lessonID), nil
}

func (l *LessonRepo) GetById(ctx context.Context, id int) (models.Lesson, error) {
	lessonIDInt64 := int64(id)
	lessonDB, err := l.db.GetLessonByID(ctx, lessonIDInt64)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Lesson{}, models.ErrNotFound
		}
		return models.Lesson{}, fmt.Errorf("get lesson by ID: %w", err)
	}
	exercisesDB, err := l.db.LessonExercisesByOrderAscPosition(ctx, lessonIDInt64)
	if err != nil {
		return models.Lesson{}, fmt.Errorf("get lessons exercises: %w", err)
	}
	lessonModel := transferToModelsLesson(lessonDB)
	fillLessonWithExercises(exercisesDB, &lessonModel)
	return lessonModel, nil
}

func (l *LessonRepo) ListByOwner(ctx context.Context, ownerID int) ([]models.Lesson, error) {
	ownerLessons, err := l.db.OwnerLessons(ctx, int64(ownerID))
	if err != nil {
		return nil, fmt.Errorf("get owner lessons by ID: %w", err)
	}
	lessonsModelArray := make([]models.Lesson, 0, len(ownerLessons))
	for _, lesson := range ownerLessons {
		lessonsModelArray = append(lessonsModelArray, transferToModelsLesson(lesson))
	}
	return lessonsModelArray, nil
}

func (l *LessonRepo) ListMarketPlace(ctx context.Context, limit int) ([]models.Lesson, error) {
	lessons, err := l.db.MarketLessonsPublicApproved(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("get lesson market place list: %w", err)
	}
	lessonsModelArray := make([]models.Lesson, 0, len(lessons))
	for _, lesson := range lessons {
		lessonsModelArray = append(lessonsModelArray, transferToModelsLesson(lesson))
	}
	return lessonsModelArray, nil
}

func (l *LessonRepo) SaveCopy(ctx context.Context, originalLessonID, copyOwnerID int) (int, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin save copy transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := l.db.WithTx(tx)

	lesson, err := q.LessonPublicApproved(ctx, int64(originalLessonID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, models.ErrNotFound
		}
		return 0, fmt.Errorf("get public approved lesson by original lesson ID: %w", err)
	}
	newCopiedLesson := db.InsertLessonParams{
		OwnerID:      int64(copyOwnerID),
		Title:        lesson.Title,
		Explanation:  lesson.Explanation,
		Level:        lesson.Level,
		Topic:        lesson.Topic,
		Difficulty:   lesson.Difficulty,
		Focus:        lesson.Focus,
		Size:         lesson.Size,
		Visibility:   string(models.VisibilityPrivate),
		CopiedFromID: pgtype.Int8{Int64: int64(originalLessonID), Valid: true},
	}
	copiedID, err := q.InsertLesson(ctx, newCopiedLesson)
	if err != nil {
		return 0, fmt.Errorf("save copy: insert copy: %w", err)
	}
	originalExercises, err := q.LessonExercisesByOrderAscPosition(ctx, int64(originalLessonID))
	if err != nil {
		return 0, fmt.Errorf("get original lesson exercises: %w", err)
	}
	for _, e := range originalExercises {
		_, err := q.InsertExercise(ctx, db.InsertExerciseParams{
			LessonID: copiedID,
			Position: e.Position,
			Type:     e.Type,
			Content:  e.Content,
		})
		if err != nil {
			return 0, fmt.Errorf("save copy: insert exercise: %w", err)
		}
	}
	err = q.IncreaseSavesCount(ctx, int64(originalLessonID))
	if err != nil {
		return 0, fmt.Errorf("save copy: increase save count: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("save copy commit: %w", err)
	}
	return int(copiedID), nil
}
