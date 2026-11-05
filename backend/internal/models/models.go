package models

import (
	"encoding/json"
	"errors"
	"slices"
	"time"
)

type Item struct {
	Name string `json:"name" binding:"required"`
	ID   int    `json:"id"`
}

type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Role string

const (
	RoleStudent Role = "student"
	RoleAdmin   Role = "admin"
)

var ErrBadRequest = errors.New("bad request")

var ErrNotFound = errors.New("not found")

var ErrConflict = errors.New("conflict error")

var ErrInvalidCredentials = errors.New("wrong email or password")

var ErrUnauthorized = errors.New("unauthorized")

var ErrForbidden = errors.New("forbidden")

var ErrNoUserFound = errors.New("no user found")

var ErrTooManyRequests = errors.New("too many requests")

type Level string

const (
	LevelA1 Level = "A1"
	LevelA2 Level = "A2"
	LevelB1 Level = "B1"
	LevelB2 Level = "B2"
	LevelC1 Level = "C1"
)

var levels = []Level{LevelA1, LevelA2, LevelB1, LevelB2, LevelC1}

func (l Level) IsValid() bool {
	return slices.Contains(levels, l)
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

var difficulties = []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard}

func (d Difficulty) IsValid() bool {
	return slices.Contains(difficulties, d)
}

type Focus string

const (
	FocusGrammar    Focus = "grammar"
	FocusReading    Focus = "reading"
	FocusVocabulary Focus = "vocabulary"
	FocusMixed      Focus = "mixed"
)

var focuses = []Focus{FocusGrammar, FocusReading, FocusVocabulary, FocusMixed}

func (f Focus) IsValid() bool {
	return slices.Contains(focuses, f)
}

type Size string

const (
	SizeShort  Size = "short"
	SizeNormal Size = "normal"
	SizeLong   Size = "long"
)

var sizes = []Size{SizeShort, SizeNormal, SizeLong}

func (s Size) IsValid() bool {
	return slices.Contains(sizes, s)
}

type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

type ExerciseType string

const (
	ExerciseTypeChoice    ExerciseType = "choice"
	ExerciseTypeGap       ExerciseType = "gap"
	ExerciseTypeInflect   ExerciseType = "inflect"
	ExerciseTypeMatch     ExerciseType = "match"
	ExerciseTypeOrder     ExerciseType = "order"
	ExerciseTypeTrueFalse ExerciseType = "true_false"
)

var exerciseTypes = []ExerciseType{ExerciseTypeChoice, ExerciseTypeGap, ExerciseTypeInflect, ExerciseTypeMatch, ExerciseTypeOrder, ExerciseTypeTrueFalse}

func (e ExerciseType) IsValid() bool {
	return slices.Contains(exerciseTypes, e)
}

type ModerationStatus string

const (
	ModerationStatusPending  ModerationStatus = "pending"
	ModerationStatusApproved ModerationStatus = "approved"
	ModerationStatusRejected ModerationStatus = "rejected"
	ModerationStatusHidden   ModerationStatus = "hidden"
)

type Lesson struct {
	ID               int               `json:"id"`
	OwnerID          int               `json:"owner_id"`
	Title            string            `json:"title"`
	Explanation      string            `json:"explanation"`
	Level            Level             `json:"level"`
	Topic            string            `json:"topic"`
	Difficulty       Difficulty        `json:"difficulty"`
	Focus            Focus             `json:"focus"`
	Size             Size              `json:"size"`
	Visibility       Visibility        `json:"visibility"`
	Exercises        []Exercise        `json:"exercises"`
	CopiedFromID     *int              `json:"copied_from_id"`
	ModerationStatus *ModerationStatus `json:"moderation_status"`
	SavesCount       int               `json:"saves_count"`
	CreatedAt        time.Time         `json:"created_at"`
}
type Exercise struct {
	ID        int             `json:"id"`
	LessonID  int             `json:"lesson_id"`
	Position  int             `json:"position"`
	Type      ExerciseType    `json:"type"`
	Content   json.RawMessage `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
}
