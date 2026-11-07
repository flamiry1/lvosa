package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/flamiry1/lvosa/backend/internal/models"

	"github.com/gin-gonic/gin"
)

type Service interface {
	List(ctx context.Context) ([]models.Item, error)
	Get(ctx context.Context, id int) (models.Item, error)
	Create(ctx context.Context, item models.Item) (models.Item, error)
	Update(ctx context.Context, item models.Item, id int) error
	Delete(ctx context.Context, id int) error
}

type GenerationCreator interface {
	Create(ctx context.Context, req models.GenerationRequest) (int, error)
	Get(ctx context.Context, id, userID int) (models.GenerationRequest, error)
}

func handleError(err error, c *gin.Context) {
	if errors.Is(err, models.ErrBadRequest) {
		c.JSON(http.StatusBadRequest, gin.H{"status": models.ErrBadRequest.Error(), "message": err.Error()})
		return
	}
	if errors.Is(err, models.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"status": models.ErrNotFound.Error()})
		return
	}
	if errors.Is(err, models.ErrConflict) {
		c.JSON(http.StatusConflict, gin.H{"status": models.ErrConflict.Error()})
		return
	}
	if errors.Is(err, models.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": models.ErrInvalidCredentials.Error()})
		return
	}
	if errors.Is(err, models.ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": models.ErrUnauthorized.Error()})
		return
	}
	if errors.Is(err, models.ErrForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"status": models.ErrForbidden.Error()})
		return
	}
	if errors.Is(err, models.ErrTooManyRequests) {
		c.JSON(http.StatusTooManyRequests, gin.H{"status": models.ErrTooManyRequests.Error()})
		return
	}
	slog.Error("internal error", "err", err)
	c.JSON(http.StatusInternalServerError, gin.H{"status": "internal server error"})
}

type Handler struct {
	svc  Service
	gSvc GenerationCreator
}

func NewItemHandler(s Service, g GenerationCreator) *Handler {
	return &Handler{svc: s, gSvc: g}
}

func (h *Handler) CreateGenerationRequest(c *gin.Context) {
	var in struct {
		Level      string `json:"level" binding:"required"`
		Topic      string `json:"topic" binding:"required"`
		Difficulty string `json:"difficulty" binding:"required"`
		Focus      string `json:"focus" binding:"required"`
		Size       string `json:"size" binding:"required"`
	}
	user, err := getUserFromContext(c)
	if err != nil {
		handleError(err, c)
		return
	}
	err = c.ShouldBindJSON(&in)
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	convertedIn := models.GenerationRequest{
		UserID:     user.ID,
		Level:      models.Level(in.Level),
		Topic:      in.Topic,
		Difficulty: models.Difficulty(in.Difficulty),
		Focus:      models.Focus(in.Focus),
		Size:       models.Size(in.Size),
	}
	ctx := c.Request.Context()
	id, err := h.gSvc.Create(ctx, convertedIn)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted", "id": id})

}

func (h *Handler) GetGenerationRequest(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		err = fmt.Errorf("%w: %w", err, models.ErrBadRequest)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	user, err := getUserFromContext(c)
	if err != nil {
		handleError(err, c)
		return
	}
	req, err := h.gSvc.Get(ctx, id, user.ID)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status, "lesson_id": req.LessonID, "error": req.Error})
}

func (h *Handler) RegisterRouters(r *gin.Engine, authMiddleWare, checkRoleMiddleWare gin.HandlerFunc) {
	authRoleProtected := r.Group("", authMiddleWare, checkRoleMiddleWare)
	{
		authRoleProtected.POST("/item", h.createItem)
		authRoleProtected.PUT("/item/:id", h.updateItem)
		authRoleProtected.DELETE("/item/:id", h.deleteItem)
	}
	authProtected := r.Group("", authMiddleWare)
	{
		authProtected.POST("/generations", h.CreateGenerationRequest)
		authProtected.GET("/generations/:id", h.GetGenerationRequest)
	}
	r.GET("/item/:id", h.getItem)
	r.GET("/items", h.allItems)
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
