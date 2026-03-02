// handler/simulate.go
// 模擬器 HTTP 處理器層
// 處理搶票模擬啟動與全域統計查詢的 HTTP 請求。
// 模擬操作委派給 SimulatorService 處理併發模擬邏輯。

package handler

import (
	"go_api_01/model"
	"go_api_01/service"
	"go_api_01/store"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SimulateHandler 模擬器 HTTP 處理器
// 封裝模擬啟動與統計查詢的路由處理函數
type SimulateHandler struct {
	// simulatorService 模擬器業務服務
	simulatorService *service.SimulatorService
	// store 記憶體存儲實例（用於取得全域統計）
	store *store.Store
}

// NewSimulateHandler 建立新的模擬器處理器實例
func NewSimulateHandler(ss *service.SimulatorService, s *store.Store) *SimulateHandler {
	return &SimulateHandler{
		simulatorService: ss,
		store:            s,
	}
}

// StartSimulation 啟動搶票模擬的 HTTP 處理函數
// POST /api/simulate
// 請求 Body: {"event_id": "evt-1", "concurrency": 500, "per_user": 1}
// 回應: APIResponse 包含模擬結果統計
func (h *SimulateHandler) StartSimulation(c *gin.Context) {
	var req model.SimulateRequest

	// 綁定並驗證請求參數
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    1,
			Message: "請求參數錯誤: " + err.Error(),
		})
		return
	}

	// 呼叫 service 層執行模擬
	result, err := h.simulatorService.RunSimulation(&req)
	if err != nil {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "模擬完成",
		Data:    result,
	})
}

// GetStats 查詢全域統計的 HTTP 處理函數
// GET /api/stats
// 回應: APIResponse 包含全域統計資訊
func (h *SimulateHandler) GetStats(c *gin.Context) {
	stats := h.store.GetStats()

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "查詢成功",
		Data:    stats,
	})
}
