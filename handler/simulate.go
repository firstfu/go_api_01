// handler/simulate.go
// 模擬器 HTTP 處理器層
// 處理搶票模擬啟動、SSE 串流模擬與全域統計查詢的 HTTP 請求。
// StartSimulation 提供傳統 JSON 回應，StreamSimulation 提供 SSE 即時進度推送。
// 模擬操作委派給 SimulatorService 處理併發模擬邏輯。

package handler

import (
	"go_api_01/model"
	"go_api_01/service"
	"go_api_01/store"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SimulateHandler 模擬器 HTTP 處理器
// 封裝模擬啟動、SSE 串流與統計查詢的路由處理函數
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
// 此端點不使用 SSE，progressCh 傳 nil（僅 console log 輸出進度）
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

	// 呼叫 service 層執行模擬（progressCh 為 nil，僅 console log）
	result, err := h.simulatorService.RunSimulation(&req, nil)
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

// StreamSimulation 以 SSE 串流方式執行搶票模擬
// GET /api/simulate/stream?event_id=xxx&concurrency=N&per_user=N
// 透過 Server-Sent Events 即時推送模擬進度至前端
// 使用 query params 因為 EventSource API 僅支援 GET 請求
func (h *SimulateHandler) StreamSimulation(c *gin.Context) {
	// 從 query params 解析參數
	eventID := c.Query("event_id")
	if eventID == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    1,
			Message: "缺少 event_id 參數",
		})
		return
	}

	concurrency, err := strconv.Atoi(c.DefaultQuery("concurrency", "100"))
	if err != nil || concurrency <= 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    1,
			Message: "concurrency 參數無效",
		})
		return
	}

	perUser, err := strconv.Atoi(c.DefaultQuery("per_user", "1"))
	if err != nil || perUser <= 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    1,
			Message: "per_user 參數無效",
		})
		return
	}

	req := &model.SimulateRequest{
		EventID:     eventID,
		Concurrency: concurrency,
		PerUser:     perUser,
	}

	// 建立 progressCh，容量 11（最多 10 次進度 + 1 次完成）
	progressCh := make(chan model.ProgressEvent, 11)

	// 設定 SSE 標頭
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 在背景 goroutine 中執行模擬
	resultCh := make(chan *model.SimulationResult, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := h.simulatorService.RunSimulation(req, progressCh)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// 串流推送進度事件
	c.Stream(func(w io.Writer) bool {
		select {
		case progress, ok := <-progressCh:
			if !ok {
				// progressCh 已關閉，等待最終結果
				select {
				case result := <-resultCh:
					c.SSEvent("result", result)
				case err := <-errCh:
					c.SSEvent("error", gin.H{"message": err.Error()})
				}
				return false
			}
			c.SSEvent("progress", progress)
			return true
		case err := <-errCh:
			c.SSEvent("error", gin.H{"message": err.Error()})
			return false
		}
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
