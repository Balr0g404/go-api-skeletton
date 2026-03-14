#!/usr/bin/env bash
# scaffold.sh — Generate boilerplate for a new resource module.
#
# Usage:
#   ./scripts/scaffold.sh <name>
#   make scaffold NAME=post
#
# Example:
#   make scaffold NAME=post
#
# Generated files:
#   internal/models/<name>.go
#   internal/repositories/<name>.go
#   internal/services/<name>.go
#   internal/handlers/<name>.go
#
# After generation:
#   1. Add your fields to internal/models/<name>.go
#   2. Add missing methods to the repository and service
#   3. Register routes in internal/router/router.go
#   4. Wire the handler in cmd/server/main.go

set -euo pipefail

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "Usage: $0 <name>"
  echo "  Example: $0 post"
  exit 1
fi

NAME="$1"
# Capitalize first letter for exported Go identifiers.
NAME_UPPER="$(tr '[:lower:]' '[:upper:]' <<< "${NAME:0:1}")${NAME:1}"
MODULE="$(grep '^module ' go.mod | awk '{print $2}')"

echo "→ Scaffolding module: $NAME_UPPER (module: $MODULE)"

# ── Helpers ──────────────────────────────────────────────────────────────────

write_file() {
  local path="$1"
  if [ -f "$path" ]; then
    echo "  SKIP  $path (already exists)"
    return
  fi
  mkdir -p "$(dirname "$path")"
  cat > "$path"
  echo "  CREATE $path"
}

# ── Model ─────────────────────────────────────────────────────────────────────

write_file "internal/models/${NAME}.go" <<GO
package models

import "gorm.io/gorm"

type ${NAME_UPPER} struct {
	gorm.Model
	// TODO: add fields
}

type ${NAME_UPPER}Response struct {
	ID uint \`json:"id"\`
	// TODO: add response fields
}

func (m *${NAME_UPPER}) ToResponse() ${NAME_UPPER}Response {
	return ${NAME_UPPER}Response{
		ID: m.ID,
	}
}
GO

# ── Repository ────────────────────────────────────────────────────────────────

write_file "internal/repositories/${NAME}.go" <<GO
package repositories

import (
	"${MODULE}/internal/models"

	"gorm.io/gorm"
)

type ${NAME_UPPER}Repository struct {
	db *gorm.DB
}

func New${NAME_UPPER}Repository(db *gorm.DB) *${NAME_UPPER}Repository {
	return &${NAME_UPPER}Repository{db: db}
}

func (r *${NAME_UPPER}Repository) Create(m *models.${NAME_UPPER}) error {
	return r.db.Create(m).Error
}

func (r *${NAME_UPPER}Repository) FindByID(id uint) (*models.${NAME_UPPER}, error) {
	var m models.${NAME_UPPER}
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *${NAME_UPPER}Repository) Update(m *models.${NAME_UPPER}) error {
	return r.db.Save(m).Error
}

func (r *${NAME_UPPER}Repository) Delete(id uint) error {
	return r.db.Delete(&models.${NAME_UPPER}{}, id).Error
}
GO

# ── Service ───────────────────────────────────────────────────────────────────

write_file "internal/services/${NAME}.go" <<GO
package services

import (
	"errors"

	"${MODULE}/internal/models"
)

var Err${NAME_UPPER}NotFound = errors.New("${NAME} not found")

// ${NAME_UPPER}Repository is the data-access interface consumed by ${NAME_UPPER}Service.
type ${NAME_UPPER}Repository interface {
	Create(m *models.${NAME_UPPER}) error
	FindByID(id uint) (*models.${NAME_UPPER}, error)
	Update(m *models.${NAME_UPPER}) error
	Delete(id uint) error
}

type ${NAME_UPPER}Service struct {
	repo ${NAME_UPPER}Repository
}

func New${NAME_UPPER}Service(repo ${NAME_UPPER}Repository) *${NAME_UPPER}Service {
	return &${NAME_UPPER}Service{repo: repo}
}

func (s *${NAME_UPPER}Service) Create(m *models.${NAME_UPPER}) (*models.${NAME_UPPER}Response, error) {
	if err := s.repo.Create(m); err != nil {
		return nil, err
	}
	resp := m.ToResponse()
	return &resp, nil
}

func (s *${NAME_UPPER}Service) GetByID(id uint) (*models.${NAME_UPPER}Response, error) {
	m, err := s.repo.FindByID(id)
	if err != nil {
		return nil, Err${NAME_UPPER}NotFound
	}
	resp := m.ToResponse()
	return &resp, nil
}

func (s *${NAME_UPPER}Service) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return Err${NAME_UPPER}NotFound
	}
	return s.repo.Delete(id)
}
GO

# ── Handler ───────────────────────────────────────────────────────────────────

write_file "internal/handlers/${NAME}.go" <<GO
package handlers

import (
	"net/http"
	"strconv"

	"${MODULE}/internal/models"
	"${MODULE}/internal/services"
	"${MODULE}/pkg/response"

	"github.com/gin-gonic/gin"
)

type ${NAME_UPPER}Handler struct {
	svc *services.${NAME_UPPER}Service
}

func New${NAME_UPPER}Handler(svc *services.${NAME_UPPER}Service) *${NAME_UPPER}Handler {
	return &${NAME_UPPER}Handler{svc: svc}
}

// Create${NAME_UPPER} godoc
// @Summary      Create a ${NAME}
// @Tags         ${NAME_UPPER}
// @Accept       json
// @Produce      json
// @Param        body  body      models.${NAME_UPPER}  true  "${NAME_UPPER} payload"
// @Success      201   {object}  response.APIResponse{data=models.${NAME_UPPER}Response}
// @Failure      400   {object}  response.APIResponse
// @Failure      500   {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /${NAME}s [post]
func (h *${NAME_UPPER}Handler) Create(c *gin.Context) {
	var m models.${NAME_UPPER}
	if err := c.ShouldBindJSON(&m); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.Create(&m)
	if err != nil {
		response.InternalError(c)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
}

// Get${NAME_UPPER} godoc
// @Summary      Get a ${NAME} by ID
// @Tags         ${NAME_UPPER}
// @Produce      json
// @Param        id   path      int  true  "${NAME_UPPER} ID"
// @Success      200  {object}  response.APIResponse{data=models.${NAME_UPPER}Response}
// @Failure      404  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /${NAME}s/{id} [get]
func (h *${NAME_UPPER}Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	result, err := h.svc.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "${NAME} not found")
		return
	}
	response.OK(c, result)
}

// Delete${NAME_UPPER} godoc
// @Summary      Delete a ${NAME}
// @Tags         ${NAME_UPPER}
// @Produce      json
// @Param        id   path      int  true  "${NAME_UPPER} ID"
// @Success      200  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /${NAME}s/{id} [delete]
func (h *${NAME_UPPER}Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		response.NotFound(c, "${NAME} not found")
		return
	}
	response.Message(c, "${NAME} deleted")
}
GO

# ── Summary ───────────────────────────────────────────────────────────────────

echo ""
echo "✓ Module '$NAME_UPPER' scaffolded. Next steps:"
echo ""
echo "  1. Add fields to internal/models/${NAME}.go"
echo "  2. Create a migration:  make migrate-create NAME=create_${NAME}s_table"
echo "  3. Wire in cmd/server/main.go:"
echo "       ${NAME}Repo    := repositories.New${NAME_UPPER}Repository(db)"
echo "       ${NAME}Service := services.New${NAME_UPPER}Service(${NAME}Repo)"
echo "       ${NAME}Handler := handlers.New${NAME_UPPER}Handler(${NAME}Service)"
echo "  4. Register routes in internal/router/router.go:"
echo "       ${NAME}s := api.Group(\"/${NAME}s\")"
echo "       ${NAME}s.Use(middleware.AuthRequired(jwtManager, authService))"
echo "       {"
echo "           ${NAME}s.POST(\"/\",    ${NAME}Handler.Create)"
echo "           ${NAME}s.GET(\"/:id\", ${NAME}Handler.GetByID)"
echo "           ${NAME}s.DELETE(\"/:id\", ${NAME}Handler.Delete)"
echo "       }"
echo "  5. Run: make swagger"
