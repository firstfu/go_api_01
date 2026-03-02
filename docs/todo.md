# Go 搶票模擬系統 — 開發待辦清單

## 實作進度

- ~~Step 1: 專案初始化（go mod init + 安裝 Gin + 建立目錄結構）~~
- ~~Step 2: model/model.go — 定義所有資料結構~~
- ~~Step 3: store/store.go — 記憶體存儲（RWMutex + atomic）~~
- ~~Step 4: service/event.go — 活動管理業務邏輯~~
- ~~Step 5: service/ticket.go — 搶票核心邏輯（Mutex 鎖）~~
- ~~Step 6: service/simulator.go — 併發模擬器（goroutine + channel）~~
- ~~Step 7: handler/event.go — 活動 HTTP 處理器~~
- ~~Step 8: handler/ticket.go — 搶票 HTTP 處理器~~
- ~~Step 9: handler/simulate.go — 模擬 HTTP 處理器~~
- ~~Step 10: router/router.go — 路由設定與 CORS 中間件~~
- ~~Step 11: main.go — 程式進入點，組裝所有依賴~~
- ~~Step 12: docs/prd.md + docs/todo.md — 產品需求文件與待辦清單~~
- ~~Step 13: store/store.go — 新增 GetEventSnapshot() 方法~~
- ~~Step 14: cmd/report/main.go — 併發壓力測試報告生成器~~
- ~~Step 15: 執行報告生成器，產出 docs/report.md~~
- ~~Step 16: model/model.go — 新增 ProgressEvent 進度事件結構~~
- ~~Step 17: service/simulator.go — 重構 RunSimulation，加入 reporter goroutine 進度匯報~~
- ~~Step 18: handler/simulate.go — 新增 StreamSimulation SSE 串流端點~~
- ~~Step 19: router/router.go — 註冊 GET /api/simulate/stream 路由~~
- ~~Step 20: web/simulate.html — 前端模擬操作頁面（SSE 即時進度）~~
