// handler/ticket.go
// 搶票 HTTP 處理器層
// 處理搶票與訂單查詢的 HTTP 請求。
// 搶票操作委派給 TicketService 處理併發控制，此層僅負責 HTTP 協議處理。

package handler

import (
	"go_api_01/model"
	"go_api_01/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TicketHandler 搶票 HTTP 處理器
// 封裝搶票與訂單查詢的路由處理函數
type TicketHandler struct {
	// ticketService 搶票業務服務
	ticketService *service.TicketService
}

// NewTicketHandler 建立新的搶票處理器實例
func NewTicketHandler(ts *service.TicketService) *TicketHandler {
	return &TicketHandler{ticketService: ts}
}

// GrabTicket 搶票的 HTTP 處理函數
// POST /api/events/:id/grab
// 請求 Body: {"user_id": "user-001", "quantity": 2}
// 回應: APIResponse 包含搶票結果
func (h *TicketHandler) GrabTicket(c *gin.Context) {
	eventID := c.Param("id")

	var req model.GrabRequest
	// 綁定並驗證請求參數
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    1,
			Message: "請求參數錯誤: " + err.Error(),
		})
		return
	}

	// 呼叫 service 層執行搶票
	result, err := h.ticketService.GrabTicket(eventID, req.UserID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    1,
			Message: result.Message,
			Data:    result,
		})
		return
	}

	// 根據搶票結果回應不同的 HTTP 狀態碼
	if result.Success {
		c.JSON(http.StatusOK, model.APIResponse{
			Code:    0,
			Message: "搶票成功",
			Data:    result,
		})
	} else {
		c.JSON(http.StatusConflict, model.APIResponse{
			Code:    1,
			Message: result.Message,
			Data:    result,
		})
	}
}

// GetOrders 查詢訂單的 HTTP 處理函數
// GET /api/events/:id/orders
// 回應: APIResponse 包含訂單列表
func (h *TicketHandler) GetOrders(c *gin.Context) {
	eventID := c.Param("id")

	// 呼叫 service 層查詢訂單
	orders := h.ticketService.GetOrdersByEvent(eventID)

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "查詢成功",
		Data: gin.H{
			"event_id":    eventID,
			"total_count": len(orders),
			"orders":      orders,
		},
	})
}
