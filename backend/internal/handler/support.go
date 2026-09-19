package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/internal/service"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type SupportHandler struct {
	auth  *service.AuthService
	audit *service.AuditService
}

func NewSupportHandler(auth *service.AuthService, audit *service.AuditService) *SupportHandler {
	return &SupportHandler{auth: auth, audit: audit}
}

func (h *SupportHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": "lng-boiloff-gas-balance", "status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

func (h *SupportHandler) Ready(c *gin.Context) {
	ctx, cancel := contextWithTimeout(c, 2*time.Second)
	defer cancel()
	if err := h.auth.Ready(ctx); err != nil {
		api.Fail(c, api.NewError(503, "DATABASE_NOT_READY", "数据库连接尚未就绪"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"database": "ok", "status": "ready"})
}

func (h *SupportHandler) Login(c *gin.Context) {
	var request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8,max=128"`
	}
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.auth.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, result)
}

func (h *SupportHandler) Me(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		return
	}
	user, err := h.auth.Me(c.Request.Context(), actor.UserID)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Success(c, http.StatusOK, user)
}

func (h *SupportHandler) ListAudits(c *gin.Context) {
	page, pageSize := pagination(c)
	filter := repository.AuditFilter{
		ActorEmail: strings.TrimSpace(c.Query("actor_email")), Action: strings.TrimSpace(c.Query("action")),
		EntityType: strings.TrimSpace(c.Query("entity_type")), Page: page, PageSize: pageSize,
	}
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			api.Fail(c, api.NewError(400, "INVALID_TIME_FILTER", "from 必须是 RFC3339 时间"))
			return
		}
		filter.From = &parsed
	}
	if value := c.Query("to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			api.Fail(c, api.NewError(400, "INVALID_TIME_FILTER", "to 必须是 RFC3339 时间"))
			return
		}
		filter.To = &parsed
	}
	events, total, err := h.audit.List(c.Request.Context(), filter)
	if err != nil {
		api.Fail(c, err)
		return
	}
	api.Page(c, events, page, pageSize, total)
}

func bindJSON(c *gin.Context, destination any) bool {
	if err := c.ShouldBindJSON(destination); err != nil {
		api.Fail(c, api.WithDetails(api.NewError(400, "INVALID_REQUEST", "请求字段格式或取值不符合要求"), map[string]any{"validation": err.Error()}))
		return false
	}
	return true
}

func parseID(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || value == 0 {
		api.Fail(c, api.NewError(400, "INVALID_ID", "资源 ID 必须是正整数"))
		return 0, false
	}
	return uint(value), true
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func optionalTimeQuery(c *gin.Context, name string) (*time.Time, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		api.Fail(c, api.WithDetails(api.NewError(400, "INVALID_TIME_FILTER", "时间筛选必须使用 RFC3339 格式"), map[string]any{"field": name}))
		return nil, false
	}
	utc := parsed.UTC()
	return &utc, true
}

func actorFromContext(c *gin.Context) (repository.Actor, bool) {
	userValue, userOK := c.Get("user_id")
	emailValue, emailOK := c.Get("email")
	roleValue, roleOK := c.Get("role")
	userID, validUser := userValue.(uint)
	email, validEmail := emailValue.(string)
	role, validRole := roleValue.(string)
	if !userOK || !emailOK || !roleOK || !validUser || !validEmail || !validRole {
		api.Fail(c, api.NewError(401, "AUTH_REQUIRED", "登录上下文无效，请重新登录"))
		return repository.Actor{}, false
	}
	return repository.Actor{UserID: userID, Email: email, Role: role, RequestID: api.RequestID(c)}, true
}

func contextWithTimeout(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
}
