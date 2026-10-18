package models

import (
	"errors"
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
