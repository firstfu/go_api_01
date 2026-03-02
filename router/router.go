// router/router.go
// 路由設定層
// 負責註冊所有 API 路由與中間件。
// 使用 gin.Default() 內建 Logger（請求日誌）和 Recovery（panic 恢復）中間件。
// 額外加入 CORS 中間件，允許跨域請求。

package router

import (
	"go_api_01/handler"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter 設定並回傳 Gin 路由引擎
// 參數為各 handler 實例，由 main.go 注入
func SetupRouter(
	eventHandler *handler.EventHandler,
	ticketHandler *handler.TicketHandler,
	simulateHandler *handler.SimulateHandler,
) *gin.Engine {
	// 建立 Gin 引擎（含 Logger + Recovery 中間件）
	r := gin.Default()

	// 註冊 CORS 中間件，允許跨域請求
	r.Use(corsMiddleware())

	// API 路由群組
	api := r.Group("/api")
	{
		// 活動相關路由
		api.POST("/events", eventHandler.CreateEvent)
		api.GET("/events/:id", eventHandler.GetEvent)

		// 搶票相關路由
		api.POST("/events/:id/grab", ticketHandler.GrabTicket)
		api.GET("/events/:id/orders", ticketHandler.GetOrders)

		// 模擬相關路由
		api.POST("/simulate", simulateHandler.StartSimulation)
		api.GET("/stats", simulateHandler.GetStats)
	}

	return r
}

// corsMiddleware CORS 跨域中間件
// 允許所有來源的跨域請求，方便前端開發與測試
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// OPTIONS 預檢請求直接回應 204
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
