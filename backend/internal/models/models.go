package models

import "errors"

type Item struct {
	Name string `json:"name" binding:"required"`
	ID   int    `json:"id"`
}

type Person struct {
	Name       string `json:"name" binding:"required"`
	Occupation string `json:"occupation" binding:"required"`
}

var ErrBadRequest = errors.New("bad request")

var ErrNotFound = errors.New("not found")

var ErrConflict = errors.New("conflict error")
