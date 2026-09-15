package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/transport/http/response"
)

// --- 时光 ---

// ListMoments 列出时光事件。
func (h *Handler) ListMoments(c *gin.Context) {
	list, err := h.asset.Moments(c.Request.Context(), c.Param("familyId"), mustUserID(c), 200)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// CreateMoment 创建时光事件。
func (h *Handler) CreateMoment(c *gin.Context) {
	var in app.MomentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	m, err := h.asset.CreateMoment(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, m)
}

// --- 库存 ---

// ListInventory 列出库存。
func (h *Handler) ListInventory(c *gin.Context) {
	list, err := h.asset.Inventory(c.Request.Context(), c.Param("familyId"), mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// CreateInventoryItem 创建库存项。
func (h *Handler) CreateInventoryItem(c *gin.Context) {
	var in app.InventoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	item, err := h.asset.CreateInventory(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, item)
}

// --- 账目 ---

// ListExpenses 列出支出。查询参数 month=YYYY-MM。
func (h *Handler) ListExpenses(c *gin.Context) {
	list, err := h.asset.Expenses(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("month"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// CreateExpense 创建支出。
func (h *Handler) CreateExpense(c *gin.Context) {
	var in app.ExpenseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	e, err := h.asset.CreateExpense(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, e)
}

// --- AI ---

// AIParseRequest AI 解析请求体。
type AIParseRequest struct {
	Input string `json:"input" binding:"required"`
}

// ParseAI 把自然语言解析成结构化记录。
func (h *Handler) ParseAI(c *gin.Context) {
	var req AIParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}
	session, err := h.ai.Parse(c.Request.Context(), c.Param("familyId"), mustUserID(c), req.Input)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, session)
}

// AISummary 生成当日摘要。
func (h *Handler) AISummary(c *gin.Context) {
	rep, err := h.ai.Summary(c.Request.Context(), c.Param("familyId"), mustUserID(c))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, rep)
}
