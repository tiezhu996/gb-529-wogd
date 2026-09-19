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

type BalanceHandler struct {
	service *service.BalanceService
}

func NewBalanceHandler(balanceService *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{service: balanceService}
}

func (h *BalanceHandler) List(c *gin.Context) {
	page, pageSize := pagination(c)
	tankID, _ := strconv.ParseUint(c.Query("tank_id"), 10, 32)
	filter := repository.BalanceFilter{TankID: uint(tankID), Status: c.Query("status"), Page: page, PageSize: pageSize}
	items, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, items, page, pageSize, total)
}

func (h *BalanceHandler) Get(c *gin.Context) {
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

func (h *BalanceHandler) Run(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.RunBalanceRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Run(c.Request.Context(), request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusCreated, item)
}

func (h *BalanceHandler) Submit(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.SubmitBalanceRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Submit(c.Request.Context(), id, request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}

func (h *BalanceHandler) Review(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.ReviewBalanceRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Review(c.Request.Context(), id, request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}

func (h *BalanceHandler) Invalidate(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	var request dto.InvalidateBalanceRequest
	if !bindJSON(c, &request) {
		return
	}
	item, err := h.service.Invalidate(c.Request.Context(), id, request, actor)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, item)
}

func (h *BalanceHandler) Uncertainty(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.Uncertainty(c.Request.Context(), id)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, result)
}
