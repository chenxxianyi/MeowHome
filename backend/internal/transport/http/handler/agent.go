package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/meowhome/backend/internal/app"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/transport/http/response"
)

func (h *Handler) ListAgentMessages(c *gin.Context) {
	q := app.AgentMessageListQuery{Type: c.Query("type"), Status: c.Query("status"), Before: c.Query("before")}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			response.Err(c, appInvalid("limit must be an integer"))
			return
		}
		q.Limit = n
	}
	data, err := h.agent.ListMessages(c.Request.Context(), c.Param("familyId"), mustUserID(c), q)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetAgentMessage(c *gin.Context) {
	data, err := h.agent.GetMessage(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("messageId"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) EditAgentDraft(c *gin.Context) {
	var in app.AgentReminderEditRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.EditDraft(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("messageId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) ConfirmAgentDraft(c *gin.Context) {
	var in app.AgentConfirmRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.ConfirmDraft(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("messageId"), in.ExpectedVersion)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) DismissAgentMessage(c *gin.Context) {
	if err := h.agent.DismissMessage(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("messageId")); err != nil {
		response.Err(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) PatrolAgent(c *gin.Context) {
	var in app.AgentPatrolRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	if in.FamilyID == "" {
		response.Err(c, appInvalid("family_id is required"))
		return
	}
	data, err := h.agent.Patrol(c.Request.Context(), in.FamilyID, mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) ChatAgent(c *gin.Context) {
	var in app.AgentChatRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	data, err := h.agent.Chat(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
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
	data, err := h.agent.ListSessions(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("before"), limit)
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
	data, err := h.agent.ListSessionMessages(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("sessionId"), c.Query("before"), limit)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, data)
}

func queryLimit(c *gin.Context) (int, error) {
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, appInvalid("limit must be an integer")
		}
		return n, nil
	}
	return app.AgentMessageListDefaultLimit, nil
}

func appInvalid(message string) error {
	return apperr.InvalidRequest(apperr.CodeValidationFailed, message)
}
