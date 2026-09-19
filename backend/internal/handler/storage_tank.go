package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/service"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type TankHandler struct {
	service *service.TankService
}

func NewTankHandler(tankService *service.TankService) *TankHandler {
	return &TankHandler{service: tankService}
}

func (h *TankHandler) List(c *gin.Context) {
	page, pageSize := pagination(c)
	items, total, err := h.service.List(c.Request.Context(), page, pageSize, c.Query("status"))
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, items, page, pageSize, total)
}

func (h *TankHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}

func (h *TankHandler) Create(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.CreateTankRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusCreated, item)
}

func (h *TankHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.UpdateTankRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}

func (h *TankHandler) MeasurementQuality(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.MeasurementQuality(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, result)
}
