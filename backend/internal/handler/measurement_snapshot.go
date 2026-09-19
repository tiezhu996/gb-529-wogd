package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/internal/service"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type MeasurementHandler struct {
	service *service.MeasurementService
}

func NewMeasurementHandler(measurementService *service.MeasurementService) *MeasurementHandler {
	return &MeasurementHandler{service: measurementService}
}

func (h *MeasurementHandler) List(c *gin.Context) {
	page, pageSize := pagination(c)
	tankID, _ := strconv.ParseUint(c.Query("tank_id"), 10, 32)
	from, ok := optionalTimeQuery(c, "from")
	if !ok {
		return
	}
	to, ok := optionalTimeQuery(c, "to")
	if !ok {
		return
	}
	filter := repository.MeasurementFilter{
		TankID: uint(tankID), Quality: c.Query("quality"), From: from, To: to,
		Page: page, PageSize: pageSize,
	}
	items, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, items, page, pageSize, total)
}

func (h *MeasurementHandler) Get(c *gin.Context) {
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

func (h *MeasurementHandler) Create(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.CreateMeasurementRequest
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
