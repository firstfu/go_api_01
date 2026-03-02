// handler/event.go
// 活動 HTTP 處理器層
// 處理活動相關的 HTTP 請求，包含建立活動與查詢活動。
// 負責請求參數綁定、驗證，並以統一的 APIResponse 格式回應。

package handler

import (
	"go_api_01/model"
	"go_api_01/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EventHandler 活動 HTTP 處理器
// 封裝活動相關的路由處理函數
type EventHandler struct {
	// eventService 活動業務服務
	eventService *service.EventService
}

// NewEventHandler 建立新的活動處理器實例
func NewEventHandler(es *service.EventService) *EventHandler {
	return &EventHandler{eventService: es}
}

// CreateEvent 建立活動的 HTTP 處理函數
// POST /api/events
// 請求 Body: {"name": "活動名稱", "description": "描述", "total_tickets": 100, "price": 2800}
// 回應: APIResponse 包含建立的活動資料
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var event model.Event

	// 綁定並驗證請求參數
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    1,
			Message: "請求參數錯誤: " + err.Error(),
		})
		return
	}

	// 呼叫 service 層建立活動
	created := h.eventService.CreateEvent(&event)

	c.JSON(http.StatusCreated, model.APIResponse{
		Code:    0,
		Message: "活動建立成功",
		Data:    created,
	})
}

// GetEvent 查詢活動的 HTTP 處理函數
// GET /api/events/:id
// 回應: APIResponse 包含活動資料
func (h *EventHandler) GetEvent(c *gin.Context) {
	id := c.Param("id")

	// 呼叫 service 層查詢活動
	event := h.eventService.GetEvent(id)
	if event == nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    1,
			Message: "活動不存在",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "查詢成功",
		Data:    event,
	})
}
