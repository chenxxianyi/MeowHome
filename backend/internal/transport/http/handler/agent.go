package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/meowhome/backend/internal/app"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/transport/http/response"
)

const maxAgentJSONBytes int64 = 16 << 10

func bindAgentJSON(c *gin.Context, dst any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAgentJSONBytes)
	return c.ShouldBindJSON(dst)
}

func (h *Handler) ListAgentMessages(c *gin.Context) {
	q := app.AgentMessageListQuery{Type: c.Query("type"), Status: c.Query("status"), Before: c.Query("before")}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > app.AgentMessageListMaxLimit {
			response.Err(c, appInvalid("limit must be an integer between 1 and 50"))
			return
		}
		q.Limit = n
	}
	data, err := h.agent.ListMessages(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), q)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetAgentMessage(c *gin.Context) {
	data, err := h.agent.GetMessage(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), c.Param("messageId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) EditAgentDraft(c *gin.Context) {
	var in app.AgentReminderEditRequest
	if err := bindAgentJSON(c, &in); err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.EditDraft(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), c.Param("messageId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) ConfirmAgentDraft(c *gin.Context) {
	var in app.AgentConfirmRequest
	if err := bindAgentJSON(c, &in); err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.ConfirmDraft(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), c.Param("messageId"), in.ExpectedVersion)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) DismissAgentMessage(c *gin.Context) {
	if err := h.agent.DismissMessage(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), c.Param("messageId")); err != nil {
		response.Err(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) PatrolAgent(c *gin.Context) {
	var in app.AgentPatrolRequest
	if err := bindAgentJSON(c, &in); err != nil {
		response.Err(c, err)
		return
	}
	if in.FamilyID == "" {
		response.Err(c, appInvalid("family_id is required"))
		return
	}
	data, err := h.agent.Patrol(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), in.FamilyID, mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) ChatAgent(c *gin.Context) {
	var in app.AgentChatRequest
	if err := bindAgentJSON(c, &in); err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.Chat(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) ListAgentSessions(c *gin.Context) {
	limit, err := queryLimit(c)
	if err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.ListSessions(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), c.Query("before"), limit)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) ListAgentSessionMessages(c *gin.Context) {
	limit, err := queryLimit(c)
	if err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.ListSessionMessages(app.WithAgentRequestID(c.Request.Context(), c.GetString("request_id")), c.Param("familyId"), mustUserID(c), c.Param("sessionId"), c.Query("before"), limit)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func queryLimit(c *gin.Context) (int, error) {
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > app.AgentMessageListMaxLimit {
			return 0, appInvalid("limit must be an integer between 1 and 50")
		}
		return n, nil
	}
	return app.AgentMessageListDefaultLimit, nil
}

func appInvalid(message string) error {
	return apperr.InvalidRequest(apperr.CodeValidationFailed, message)
}
