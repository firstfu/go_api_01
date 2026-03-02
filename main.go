// main.go
// Go 搶票模擬系統 — 程式進入點
// 負責初始化所有依賴（Store → Service → Handler → Router），
// 並啟動 HTTP 伺服器監聽 :8080 埠。
//
// 架構：Handler → Service → Store（由上到下依賴）
// 併發設計：
//   - store.RWMutex：保護 map 結構安全（CRUD）
//   - ticketService.Mutex：保護搶票業務原子性
//   - atomic 操作：統計計數器的高效累加
//   - goroutine + WaitGroup + channel：模擬器 fan-out/fan-in 模式

package main

import (
	"go_api_01/handler"
	"go_api_01/router"
	"go_api_01/service"
	"go_api_01/store"
	"log"
)

func main() {
	// 初始化記憶體存儲
	s := store.NewStore()

	// 初始化業務服務層
	eventService := service.NewEventService(s)
	ticketService := service.NewTicketService(s)
	simulatorService := service.NewSimulatorService(ticketService, s)

	// 初始化 HTTP 處理器層
	eventHandler := handler.NewEventHandler(eventService)
	ticketHandler := handler.NewTicketHandler(ticketService)
	simulateHandler := handler.NewSimulateHandler(simulatorService, s)

	// 設定路由
	r := router.SetupRouter(eventHandler, ticketHandler, simulateHandler)

	// 啟動 HTTP 伺服器
	log.Println("搶票模擬系統啟動中，監聽 :8080 ...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("伺服器啟動失敗: %v", err)
	}
}
