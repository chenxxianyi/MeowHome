package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/transport/http/response"
)

// --- 待购买清单 ---

// ListShopping 列出待购买项。查询参数 status=todo|purchased|dismissed。
func (h *Handler) ListShopping(c *gin.Context) {
	list, err := h.shopping.ShoppingList(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("status"))
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}

// CreateShoppingItem 创建待购买项。
func (h *Handler) CreateShoppingItem(c *gin.Context) {
	var in app.ShoppingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	item, err := h.shopping.CreateShopping(c.Request.Context(), c.Param("familyId"), mustUserID(c), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, item)
}

// UpdateShoppingItem 修改或完成待购买项。
func (h *Handler) UpdateShoppingItem(c *gin.Context) {
	var in app.ShoppingUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	item, err := h.shopping.UpdateShopping(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("itemId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, item)
}

// DeleteShoppingItem 移除待购买项。
func (h *Handler) DeleteShoppingItem(c *gin.Context) {
	if err := h.shopping.DeleteShopping(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("itemId")); err != nil {
		response.Err(c, err)
		return
	}
	response.NoContent(c)
}

// --- 库存调整与流水 ---

// AdjustInventoryItem 手动调整库存数量并记录流水。
func (h *Handler) AdjustInventoryItem(c *gin.Context) {
	var in app.InventoryAdjustInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Err(c, err)
		return
	}
	result, err := h.shopping.AdjustInventory(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Param("itemId"), in)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.Created(c, result)
}

// ListInventoryTransactions 列出库存流水；item_id 为空表示全部，limit 默认 100。
func (h *Handler) ListInventoryTransactions(c *gin.Context) {
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	list, err := h.shopping.InventoryTransactions(c.Request.Context(), c.Param("familyId"), mustUserID(c), c.Query("item_id"), limit)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, list)
}
