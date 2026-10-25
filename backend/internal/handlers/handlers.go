package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"lvosa/backend/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Service interface {
	List(ctx context.Context) ([]models.Item, error)
	Get(ctx context.Context, id int) (models.Item, error)
	Create(ctx context.Context, item models.Item) (models.Item, error)
	Update(ctx context.Context, item models.Item, id int) error
	Delete(ctx context.Context, id int) error
}

func handleError(err error, c *gin.Context) {
	if errors.Is(err, models.ErrBadRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"status": models.ErrBadRequest.Error(), "message": err.Error()})
		return
	}
	if errors.Is(err, models.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"status": "not found"})
		return
	}
	if errors.Is(err, models.ErrConflict) {
		c.JSON(http.StatusConflict, gin.H{"status": "conflict"})
		return
	}
	if errors.Is(err, models.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
		return
	}
	slog.Error("internal error", "err", err)
	c.JSON(http.StatusInternalServerError, gin.H{"status": "internal server error"})
}

type Handler struct {
	svc Service
}

func New(s Service) *Handler {
	return &Handler{svc: s}
}

func (h *Handler) RegisterRouters(r *gin.Engine) {
	r.POST("/item", h.createItem)
	r.GET("/item/:id", h.getItem)
	r.GET("/items", h.allItems)
	r.PUT("/item/:id", h.updateItem)
	r.DELETE("/item/:id", h.deleteItem)
}

func (h *Handler) getItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	item, err := h.svc.Get(ctx, id)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) createItem(c *gin.Context) {
	var item models.Item
	err := c.ShouldBindJSON(&item)
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	item, err = h.svc.Create(ctx, item)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) allItems(c *gin.Context) {
	ctx := c.Request.Context()
	items, err := h.svc.List(ctx)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) updateItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	var item models.Item
	err = c.ShouldBindJSON(&item)
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	if err := h.svc.Update(ctx, item, id); err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "successfully updated"})
}

func (h *Handler) deleteItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	if err = h.svc.Delete(ctx, id); err != nil {
		handleError(err, c)
		return
	}
	c.Status(http.StatusNoContent)
}
