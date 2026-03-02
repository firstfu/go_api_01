# 搶票模擬進度報告功能 — 實作計畫

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 為搶票模擬加入即時進度報告功能（console log + SSE 前端推送）

**Architecture:** 搶票 goroutine 結果送入 resultCh，由 reporter goroutine 統一消費，每完成 10% 輸出 console log 並推送至 progressCh。SSE handler 讀取 progressCh 推送給前端。progressCh 可為 nil（向後相容原有 POST /api/simulate）。

**Tech Stack:** Go / Gin / SSE (Server-Sent Events) / EventSource (前端)

---

## Task 1: 新增 ProgressEvent 資料結構

**Files:**
- Modify: `model/model.go`（在檔案末尾新增）

**Step 1: 在 `model/model.go` 末尾新增 ProgressEvent struct**

```go
// ProgressEvent 模擬進度事件
// 由 reporter goroutine 產生，每完成一定比例的搶票後發送
type ProgressEvent struct {
	// Current 已完成的請求數
	Current int `json:"current"`
	// Total 總請求數
	Total int `json:"total"`
	// SuccessCount 累計成功數
	SuccessCount int `json:"success_count"`
	// FailCount 累計失敗數
	FailCount int `json:"fail_count"`
	// Remaining 活動剩餘票數
	Remaining int `json:"remaining"`
	// Percentage 完成百分比
	Percentage float64 `json:"percentage"`
	// Done 是否全部完成
	Done bool `json:"done"`
}
```

**Step 2: 確認編譯通過**

Run: `cd D:/myCodeProject/go_api_01 && go build ./...`
Expected: 無錯誤

---

## Task 2: 重構 RunSimulation — 加入 reporter goroutine

**Files:**
- Modify: `service/simulator.go`

這是核心改動。將 `RunSimulation` 的簽名改為接受 `progressCh chan<- model.ProgressEvent` 參數，並加入 reporter goroutine。

**Step 1: 修改 RunSimulation 簽名與內部邏輯**

完整替換 `RunSimulation` 方法：

```go
// RunSimulation 執行搶票模擬
// 使用 fan-out/fan-in 模式，並透過 reporter goroutine 匯報進度：
// 1. fan-out：啟動 N 個 goroutine 同時搶票，結果送入 resultCh
// 2. reporter goroutine：消費 resultCh，每完成 10% 輸出 console log + 推送 progressCh
// 3. 主流程等待 reporter 完成後彙整最終結果
//
// progressCh 可為 nil（向後相容原有 POST /api/simulate 呼叫）
func (ss *SimulatorService) RunSimulation(req *model.SimulateRequest, progressCh chan<- model.ProgressEvent) (*model.SimulationResult, error) {
	// 驗證活動是否存在
	event := ss.store.GetEvent(req.EventID)
	if event == nil {
		return nil, fmt.Errorf("活動 %s 不存在", req.EventID)
	}

	concurrency := req.Concurrency
	perUser := req.PerUser

	// 建立 buffered channel，容量等於併發數，避免 goroutine 阻塞
	resultCh := make(chan model.GrabResult, concurrency)

	// WaitGroup 追蹤所有搶票 goroutine 的完成狀態
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// fan-out：啟動 N 個 goroutine 同時搶票
	for i := 0; i < concurrency; i++ {
		go func(userIndex int) {
			defer wg.Done()
			userID := fmt.Sprintf("sim-user-%d", userIndex+1)

			// 呼叫搶票服務執行實際搶票
			result, _ := ss.ticketService.GrabTicket(req.EventID, userID, perUser)
			if result != nil {
				resultCh <- *result
			}
		}(i)
	}

	// 獨立 goroutine：等待所有搶票完成後關閉 channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// reporter goroutine：消費 resultCh，匯報進度並收集結果
	// 使用 doneCh 通知主流程 reporter 已完成
	type reporterResult struct {
		results         []model.GrabResult
		successCount    int
		failCount       int
		totalRespTime   float64
		minRespTime     float64
		maxRespTime     float64
	}

	doneCh := make(chan reporterResult, 1)

	// 計算匯報間隔：每完成 10%（至少每 1 筆）匯報一次
	reportInterval := concurrency / 10
	if reportInterval < 1 {
		reportInterval = 1
	}

	go func() {
		var rr reporterResult
		rr.minRespTime = math.MaxFloat64
		count := 0

		for result := range resultCh {
			// 收集結果
			rr.results = append(rr.results, result)
			count++

			if result.Success {
				rr.successCount++
			} else {
				rr.failCount++
			}

			rr.totalRespTime += result.ResponseTime

			if result.ResponseTime < rr.minRespTime {
				rr.minRespTime = result.ResponseTime
			}
			if result.ResponseTime > rr.maxRespTime {
				rr.maxRespTime = result.ResponseTime
			}

			// 每 N 筆匯報一次進度
			if count%reportInterval == 0 || count == concurrency {
				// 取得目前活動剩餘票數
				snapshot := ss.store.GetEventSnapshot(req.EventID)
				remaining := 0
				if snapshot != nil {
					remaining = snapshot.Remaining
				}

				pct := float64(count) / float64(concurrency) * 100

				// console log 輸出
				if count == concurrency {
					log.Printf("[模擬進度] [%d/%d] 成功: %d | 失敗: %d | 剩餘票數: %d ✓ 完成",
						count, concurrency, rr.successCount, rr.failCount, remaining)
				} else {
					log.Printf("[模擬進度] [%d/%d] 成功: %d | 失敗: %d | 剩餘票數: %d",
						count, concurrency, rr.successCount, rr.failCount, remaining)
				}

				// 推送至 progressCh（若有）
				if progressCh != nil {
					progressCh <- model.ProgressEvent{
						Current:      count,
						Total:        concurrency,
						SuccessCount: rr.successCount,
						FailCount:    rr.failCount,
						Remaining:    remaining,
						Percentage:   pct,
						Done:         count == concurrency,
					}
				}
			}
		}

		// 若無結果，將最小回應時間設為 0
		if rr.minRespTime == math.MaxFloat64 {
			rr.minRespTime = 0
		}

		doneCh <- rr
	}()

	// 等待 reporter 完成
	rr := <-doneCh

	// 若 progressCh 不為 nil，關閉它
	if progressCh != nil {
		close(progressCh)
	}

	// 計算統計數據
	totalRequests := len(rr.results)
	var successRate, avgResponseTime float64

	if totalRequests > 0 {
		successRate = float64(rr.successCount) / float64(totalRequests) * 100
		avgResponseTime = rr.totalRespTime / float64(totalRequests)
	}

	return &model.SimulationResult{
		EventID:         req.EventID,
		TotalRequests:   totalRequests,
		SuccessCount:    rr.successCount,
		FailCount:       rr.failCount,
		SuccessRate:     successRate,
		AvgResponseTime: avgResponseTime,
		MinResponseTime: rr.minRespTime,
		MaxResponseTime: rr.maxRespTime,
		Results:         rr.results,
	}, nil
}
```

**Step 2: 在 import 區加入 `"log"` 套件**

確認 import 包含 `"log"`（原本已有 `"fmt"`, `"math"`, `"sync"`）。

**Step 3: 確認編譯通過**

Run: `cd D:/myCodeProject/go_api_01 && go build ./...`
Expected: 編譯失敗 — `handler/simulate.go` 呼叫 `RunSimulation` 參數數量不匹配（預期，Task 3 修復）

---

## Task 3: 修改 handler — 向後相容 + 新增 SSE handler

**Files:**
- Modify: `handler/simulate.go`

**Step 1: 修改 StartSimulation，傳入 nil 作為 progressCh**

將第 51 行：
```go
result, err := h.simulatorService.RunSimulation(&req)
```
改為：
```go
result, err := h.simulatorService.RunSimulation(&req, nil)
```

**Step 2: 新增 StreamSimulation SSE handler**

在 `handler/simulate.go` 末尾新增：

```go
// StreamSimulation 以 SSE 串流方式執行搶票模擬
// GET /api/simulate/stream?event_id=xxx&concurrency=N&per_user=N
// 透過 Server-Sent Events 即時推送模擬進度至前端
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
```

**Step 3: 在 import 區加入 `"strconv"` 和 `"io"` 套件**

確認 import 包含 `"strconv"` 和 `"io"`。

**Step 4: 確認編譯通過**

Run: `cd D:/myCodeProject/go_api_01 && go build ./...`
Expected: 無錯誤

---

## Task 4: 註冊 SSE 路由

**Files:**
- Modify: `router/router.go`

**Step 1: 在模擬相關路由區塊新增 SSE 路由**

在 `api.GET("/stats", simulateHandler.GetStats)` 之後加入：

```go
		// SSE 模擬串流路由
		api.GET("/simulate/stream", simulateHandler.StreamSimulation)
```

**Step 2: 確認編譯通過並啟動測試**

Run: `cd D:/myCodeProject/go_api_01 && go build ./...`
Expected: 無錯誤

---

## Task 5: 新增前端模擬頁面

**Files:**
- Create: `web/simulate.html`

**Step 1: 建立 `web/simulate.html`**

使用與現有頁面（event.html）一致的賽博龐克風格，包含：
- 表單區：event_id、concurrency、per_user 輸入框 + 開始模擬按鈕
- 進度區：進度條 + 已完成/成功/失敗 即時計數
- 結果區：模擬完成後顯示完整統計（成功率、回應時間等）
- 使用 EventSource 連線 `/api/simulate/stream` 接收 SSE 事件

頁面需沿用 `event.html` 的 CSS 變數（--cyan, --magenta, --green 等）與導航列結構。

**Step 2: 在瀏覽器中開啟確認頁面渲染正常**

開啟 `http://localhost:8080/web/simulate.html`

---

## Task 6: 更新導航列 + 文件

**Files:**
- Modify: `web/event.html`（導航列加「模擬」連結）
- Modify: `web/orders.html`（導航列加「模擬」連結）
- Modify: `web/stats.html`（導航列加「模擬」連結）
- Modify: `docs/todo.md`（新增已完成項目）

**Step 1: 在三個 HTML 的導航列 `<div class="links">` 中加入模擬連結**

在 `stats` 連結後面加入：
```html
    <a href="/web/simulate.html">模擬</a>
```

**Step 2: 在 `docs/todo.md` 末尾新增已完成項目**

```markdown
- ~~Step 16: 進度報告功能 — ProgressEvent model + reporter goroutine~~
- ~~Step 17: SSE 串流端點 GET /api/simulate/stream~~
- ~~Step 18: 前端模擬頁面 web/simulate.html~~
```

---

## Task 7: 端對端驗證

**Step 1: 啟動伺服器**

Run: `cd D:/myCodeProject/go_api_01 && go run main.go`

**Step 2: 建立測試活動**

```bash
curl -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d '{"name":"測試活動","total_tickets":100,"price":500}'
```

**Step 3: 用 curl 測試 SSE 串流**

```bash
curl -N "http://localhost:8080/api/simulate/stream?event_id=evt-1&concurrency=200&per_user=1"
```

Expected: 看到多筆 `event: progress` + 最後一筆 `event: result`

**Step 4: 確認 console log 輸出**

在伺服器 terminal 中應看到：
```
[模擬進度] [20/200] 成功: 20 | 失敗: 0 | 剩餘票數: 80
...
[模擬進度] [200/200] 成功: 100 | 失敗: 100 | 剩餘票數: 0 ✓ 完成
```

**Step 5: 在瀏覽器中測試前端頁面**

開啟 `http://localhost:8080/web/simulate.html`，填入參數並點擊開始模擬。

**Step 6: 測試向後相容（原有 POST 端點）**

```bash
curl -X POST http://localhost:8080/api/simulate \
  -H "Content-Type: application/json" \
  -d '{"event_id":"evt-1","concurrency":50,"per_user":1}'
```

Expected: 回傳完整 JSON 結果（與之前行為一致）
