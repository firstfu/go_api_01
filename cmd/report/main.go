// cmd/report/main.go
// 併發壓力測試報告生成器
// 自動執行多種壓力測試場景（從低競爭到超大型活動），
// 驗證 Go 搶票系統的併發正確性（無超賣）與效能表現，
// 並輸出完整的 Markdown 報告至 docs/report.md。
// 每個測試場景使用獨立的 Store 實例，確保場景之間互不影響。

package main

import (
	"fmt"
	"go_api_01/model"
	"go_api_01/service"
	"go_api_01/store"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

// TestScenario 測試場景定義
// 描述一組壓力測試的參數配置
type TestScenario struct {
	// Name 場景名稱
	Name string
	// TotalTickets 活動總票數
	TotalTickets int
	// Concurrency 併發用戶數量
	Concurrency int
	// PerUser 每位用戶搶購的票數
	PerUser int
	// Price 票價
	Price float64
}

// ScenarioResult 單一場景的測試結果
// 包含模擬結果、超賣驗證資訊、效能指標
type ScenarioResult struct {
	// Scenario 原始場景定義
	Scenario TestScenario
	// SimResult 模擬器回傳的結果
	SimResult *model.SimulationResult
	// FinalRemaining 活動最終剩餘票數
	FinalRemaining int
	// TotalSold 實際售出總票數（根據訂單累計）
	TotalSold int
	// OrderCount 訂單總數
	OrderCount int
	// Oversold 是否超賣（三重檢查任一失敗即為 true）
	Oversold bool
	// Duration 場景執行總耗時
	Duration time.Duration
	// Throughput 吞吐量（請求數/秒）
	Throughput float64
	// P50 回應時間第 50 百分位數（毫秒）
	P50 float64
	// P95 回應時間第 95 百分位數（毫秒）
	P95 float64
	// P99 回應時間第 99 百分位數（毫秒）
	P99 float64
}

// scenarios 定義所有測試場景
var scenarios = []TestScenario{
	{Name: "低競爭", TotalTickets: 100, Concurrency: 100, PerUser: 1, Price: 500},
	{Name: "中競爭", TotalTickets: 100, Concurrency: 500, PerUser: 1, Price: 500},
	{Name: "高競爭", TotalTickets: 100, Concurrency: 1000, PerUser: 1, Price: 500},
	{Name: "超高競爭", TotalTickets: 100, Concurrency: 5000, PerUser: 1, Price: 500},
	{Name: "極限競爭", TotalTickets: 100, Concurrency: 10000, PerUser: 1, Price: 500},
	{Name: "多票搶購-低", TotalTickets: 200, Concurrency: 500, PerUser: 2, Price: 800},
	{Name: "多票搶購-高", TotalTickets: 200, Concurrency: 2000, PerUser: 2, Price: 800},
	{Name: "大型活動", TotalTickets: 1000, Concurrency: 5000, PerUser: 1, Price: 1200},
	{Name: "超大型活動", TotalTickets: 5000, Concurrency: 10000, PerUser: 1, Price: 2000},
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  Go 搶票模擬系統 — 併發壓力測試")
	fmt.Println("========================================")
	fmt.Printf("Go 版本: %s\n", runtime.Version())
	fmt.Printf("作業系統: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPU 核心數: %d\n", runtime.NumCPU())
	fmt.Printf("測試場景數: %d\n", len(scenarios))
	fmt.Println()

	// 執行所有測試場景
	results := make([]ScenarioResult, 0, len(scenarios))
	allPassed := true

	for i, scenario := range scenarios {
		fmt.Printf("[%d/%d] 執行場景: %s（票數=%d, 併發=%d, 每人=%d）...\n",
			i+1, len(scenarios), scenario.Name,
			scenario.TotalTickets, scenario.Concurrency, scenario.PerUser)

		result := runScenario(scenario)
		results = append(results, result)

		// 即時顯示結果
		if result.Oversold {
			allPassed = false
			fmt.Printf("  ❌ 超賣警告！剩餘=%d, 售出=%d, 總票=%d\n",
				result.FinalRemaining, result.TotalSold, scenario.TotalTickets)
		} else {
			fmt.Printf("  ✅ 通過 | 成功=%d, 失敗=%d, 成功率=%.1f%%, 耗時=%v\n",
				result.SimResult.SuccessCount, result.SimResult.FailCount,
				result.SimResult.SuccessRate, result.Duration.Round(time.Millisecond))
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("✅ 所有場景均通過超賣驗證！")
	} else {
		fmt.Println("❌ 存在超賣場景，請檢查併發控制邏輯！")
	}

	// 生成 Markdown 報告
	fmt.Println()
	fmt.Println("正在生成報告...")
	report := generateReport(results)

	// 寫入檔案
	err := os.WriteFile("docs/report.md", []byte(report), 0644)
	if err != nil {
		fmt.Printf("❌ 寫入報告失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 報告已生成: docs/report.md")
}

// runScenario 執行單一測試場景
// 建立獨立的 Store → TicketService → SimulatorService 服務鏈，
// 建立活動、執行模擬、驗證結果
func runScenario(scenario TestScenario) ScenarioResult {
	// 1. 建立獨立的服務實例
	s := store.NewStore()
	ts := service.NewTicketService(s)
	simSvc := service.NewSimulatorService(ts, s)

	// 2. 建立活動
	event := s.CreateEvent(&model.Event{
		Name:         fmt.Sprintf("壓測活動-%s", scenario.Name),
		Description:  fmt.Sprintf("壓力測試場景: %s", scenario.Name),
		TotalTickets: scenario.TotalTickets,
		Price:        scenario.Price,
	})

	// 3. 執行模擬
	startTime := time.Now()
	simResult, err := simSvc.RunSimulation(&model.SimulateRequest{
		EventID:     event.ID,
		Concurrency: scenario.Concurrency,
		PerUser:     scenario.PerUser,
	})
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("  ⚠️ 模擬執行錯誤: %v\n", err)
		return ScenarioResult{Scenario: scenario, Oversold: true, Duration: duration}
	}

	// 4. 讀取最終活動狀態（使用快照）
	snapshot := s.GetEventSnapshot(event.ID)
	finalRemaining := snapshot.Remaining

	// 5. 統計實際售出票數（透過訂單累加）
	orders := s.GetOrdersByEvent(event.ID)
	totalSold := 0
	for _, order := range orders {
		totalSold += order.Quantity
	}

	// 6. 超賣三重驗證
	oversold := false
	if finalRemaining < 0 {
		oversold = true // 剩餘不為負
	}
	if totalSold > scenario.TotalTickets {
		oversold = true // 售出不超過總數
	}
	if finalRemaining+totalSold != scenario.TotalTickets {
		oversold = true // 守恆等式
	}

	// 7. 計算百分位數
	p50, p95, p99 := calculatePercentiles(simResult.Results)

	// 8. 計算吞吐量
	var throughput float64
	if duration.Seconds() > 0 {
		throughput = float64(simResult.TotalRequests) / duration.Seconds()
	}

	return ScenarioResult{
		Scenario:       scenario,
		SimResult:      simResult,
		FinalRemaining: finalRemaining,
		TotalSold:      totalSold,
		OrderCount:     len(orders),
		Oversold:       oversold,
		Duration:       duration,
		Throughput:     throughput,
		P50:            p50,
		P95:            p95,
		P99:            p99,
	}
}

// calculatePercentiles 計算回應時間的百分位數
// 從所有 GrabResult 中提取 ResponseTime，排序後取 P50/P95/P99
func calculatePercentiles(results []model.GrabResult) (p50, p95, p99 float64) {
	if len(results) == 0 {
		return 0, 0, 0
	}

	// 提取所有回應時間
	times := make([]float64, len(results))
	for i, r := range results {
		times[i] = r.ResponseTime
	}

	// 排序
	sort.Float64s(times)

	n := len(times)
	p50 = times[percentileIndex(n, 50)]
	p95 = times[percentileIndex(n, 95)]
	p99 = times[percentileIndex(n, 99)]

	return p50, p95, p99
}

// percentileIndex 計算百分位數對應的陣列索引
func percentileIndex(n, percentile int) int {
	idx := (n * percentile) / 100
	if idx >= n {
		idx = n - 1
	}
	return idx
}

// generateReport 生成完整的 Markdown 報告
func generateReport(results []ScenarioResult) string {
	var sb strings.Builder

	// 1. 標題與摘要
	writeHeader(&sb, results)

	// 2. 系統架構概述
	writeArchitecture(&sb)

	// 3. 測試環境
	writeEnvironment(&sb)

	// 4. 測試結果總表
	writeResultsTable(&sb, results)

	// 5. 超賣驗證專區
	writeOversoldVerification(&sb, results)

	// 6. 效能分析
	writePerformanceAnalysis(&sb, results)

	// 7. Go 併發優勢總結
	writeConcurrencyAdvantages(&sb)

	// 8. 結論
	writeConclusion(&sb, results)

	return sb.String()
}

// writeHeader 寫入報告標題與摘要
func writeHeader(sb *strings.Builder, results []ScenarioResult) {
	totalRequests := 0
	totalSuccess := 0
	allPassed := true
	for _, r := range results {
		if r.SimResult != nil {
			totalRequests += r.SimResult.TotalRequests
			totalSuccess += r.SimResult.SuccessCount
		}
		if r.Oversold {
			allPassed = false
		}
	}

	sb.WriteString("# Go 搶票模擬系統 — 併發壓力測試報告\n\n")
	sb.WriteString(fmt.Sprintf("> 報告生成時間: %s  \n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("> Go 版本: %s  \n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("> 作業系統: %s/%s  \n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("> CPU 核心數: %d  \n\n", runtime.NumCPU()))

	sb.WriteString("## 摘要\n\n")
	sb.WriteString(fmt.Sprintf("- 測試場景數: **%d**\n", len(results)))
	sb.WriteString(fmt.Sprintf("- 總請求數: **%d**\n", totalRequests))
	sb.WriteString(fmt.Sprintf("- 總成功數: **%d**\n", totalSuccess))
	if allPassed {
		sb.WriteString("- 超賣驗證: **全部通過 (PASS)**\n")
	} else {
		sb.WriteString("- 超賣驗證: **存在失敗 (FAIL)**\n")
	}
	sb.WriteString("\n---\n\n")
}

// writeArchitecture 寫入系統架構概述
func writeArchitecture(sb *strings.Builder) {
	sb.WriteString("## 系統架構概述\n\n")
	sb.WriteString("### 分層架構\n\n")
	sb.WriteString("```\n")
	sb.WriteString("Handler（HTTP 處理）→ Service（業務邏輯）→ Store（記憶體存儲）\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### 併發控制機制\n\n")
	sb.WriteString("| 機制 | 用途 | 位置 |\n")
	sb.WriteString("|------|------|------|\n")
	sb.WriteString("| `sync.Mutex` | 搶票原子性保護（讀庫存→判斷→扣減→下單） | `service/ticket.go` |\n")
	sb.WriteString("| `sync.RWMutex` | map 讀寫保護（允許多讀單寫） | `store/store.go` |\n")
	sb.WriteString("| `sync/atomic` | 統計計數器（無鎖高效能） | `store/store.go` |\n")
	sb.WriteString("| `goroutine` | 輕量級併發執行單元 | `service/simulator.go` |\n")
	sb.WriteString("| `channel` | fan-out/fan-in 結果收集 | `service/simulator.go` |\n")
	sb.WriteString("| `sync.WaitGroup` | 等待所有 goroutine 完成 | `service/simulator.go` |\n\n")

	sb.WriteString("### 防超賣原理\n\n")
	sb.WriteString("```go\n")
	sb.WriteString("ts.mu.Lock()              // 1. 加鎖\n")
	sb.WriteString("event := ts.store.GetEvent(id)  // 2. 讀庫存\n")
	sb.WriteString("if event.Remaining < qty { ... } // 3. 判斷\n")
	sb.WriteString("event.Remaining -= qty           // 4. 扣減\n")
	sb.WriteString("ts.store.CreateOrder(order)       // 5. 下單\n")
	sb.WriteString("ts.mu.Unlock()                    // 6. 解鎖\n")
	sb.WriteString("```\n\n")
	sb.WriteString("步驟 2~5 在同一把 Mutex 內完成，確保任何時刻只有一個 goroutine 執行搶票，\n")
	sb.WriteString("從根本上杜絕了超賣的可能性。\n\n")
	sb.WriteString("---\n\n")
}

// writeEnvironment 寫入測試環境資訊
func writeEnvironment(sb *strings.Builder) {
	sb.WriteString("## 測試環境\n\n")
	sb.WriteString("| 項目 | 數值 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| Go 版本 | %s |\n", runtime.Version()))
	sb.WriteString(fmt.Sprintf("| 作業系統 | %s/%s |\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("| CPU 核心數 | %d |\n", runtime.NumCPU()))
	sb.WriteString(fmt.Sprintf("| GOMAXPROCS | %d |\n", runtime.GOMAXPROCS(0)))
	sb.WriteString("| 存儲方式 | 純記憶體（無資料庫） |\n")
	sb.WriteString("| 網路層 | 無（直接呼叫 Service 層） |\n\n")
	sb.WriteString("---\n\n")
}

// writeResultsTable 寫入測試結果總表
func writeResultsTable(sb *strings.Builder, results []ScenarioResult) {
	sb.WriteString("## 測試結果總表\n\n")
	sb.WriteString("| 場景 | 票數 | 併發 | 每人 | 成功 | 失敗 | 成功率 | 平均回應(ms) | 吞吐量(req/s) | 耗時 |\n")
	sb.WriteString("|------|------|------|------|------|------|--------|-------------|--------------|------|\n")

	for _, r := range results {
		if r.SimResult == nil {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | - | - | - | - | - | %v |\n",
				r.Scenario.Name, r.Scenario.TotalTickets,
				r.Scenario.Concurrency, r.Scenario.PerUser,
				r.Duration.Round(time.Millisecond)))
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %.1f%% | %.4f | %.0f | %v |\n",
			r.Scenario.Name,
			r.Scenario.TotalTickets,
			r.Scenario.Concurrency,
			r.Scenario.PerUser,
			r.SimResult.SuccessCount,
			r.SimResult.FailCount,
			r.SimResult.SuccessRate,
			r.SimResult.AvgResponseTime,
			r.Throughput,
			r.Duration.Round(time.Millisecond),
		))
	}
	sb.WriteString("\n---\n\n")
}

// writeOversoldVerification 寫入超賣驗證專區
func writeOversoldVerification(sb *strings.Builder, results []ScenarioResult) {
	sb.WriteString("## 超賣驗證（最關鍵指標）\n\n")
	sb.WriteString("超賣驗證使用三重檢查機制：\n\n")
	sb.WriteString("1. **剩餘不為負**: `remaining >= 0`\n")
	sb.WriteString("2. **售出不超過總數**: `totalSold <= totalTickets`\n")
	sb.WriteString("3. **守恆等式**: `remaining + totalSold == totalTickets`\n\n")

	sb.WriteString("| 場景 | 總票數 | 剩餘 | 售出 | 訂單數 | 剩餘≥0 | 售出≤總數 | 守恆等式 | 結果 |\n")
	sb.WriteString("|------|--------|------|------|--------|--------|-----------|---------|------|\n")

	allPassed := true
	for _, r := range results {
		if r.SimResult == nil {
			sb.WriteString(fmt.Sprintf("| %s | %d | - | - | - | - | - | - | ❌ ERROR |\n",
				r.Scenario.Name, r.Scenario.TotalTickets))
			allPassed = false
			continue
		}

		checkRemaining := r.FinalRemaining >= 0
		checkSold := r.TotalSold <= r.Scenario.TotalTickets
		checkConservation := r.FinalRemaining+r.TotalSold == r.Scenario.TotalTickets

		passRemaining := "PASS"
		if !checkRemaining {
			passRemaining = "FAIL"
		}
		passSold := "PASS"
		if !checkSold {
			passSold = "FAIL"
		}
		passConservation := "PASS"
		if !checkConservation {
			passConservation = "FAIL"
		}

		result := "✅ PASS"
		if r.Oversold {
			result = "❌ FAIL"
			allPassed = false
		}

		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %s | %s | %s | %s |\n",
			r.Scenario.Name,
			r.Scenario.TotalTickets,
			r.FinalRemaining,
			r.TotalSold,
			r.OrderCount,
			passRemaining,
			passSold,
			passConservation,
			result,
		))
	}

	sb.WriteString("\n")
	if allPassed {
		sb.WriteString("### ✅ 超賣驗證結論: 全部通過\n\n")
		sb.WriteString("在所有測試場景中（最高 10,000 併發），系統均未發生超賣。\n")
		sb.WriteString("Go 的 `sync.Mutex` 完美保護了搶票操作的原子性，\n")
		sb.WriteString("守恆等式 `remaining + sold == total` 在每個場景中都嚴格成立。\n\n")
	} else {
		sb.WriteString("### ❌ 超賣驗證結論: 存在失敗\n\n")
		sb.WriteString("部分場景未通過超賣驗證，請檢查併發控制邏輯。\n\n")
	}
	sb.WriteString("---\n\n")
}

// writePerformanceAnalysis 寫入效能分析
func writePerformanceAnalysis(sb *strings.Builder, results []ScenarioResult) {
	sb.WriteString("## 效能分析\n\n")

	// 回應時間分析
	sb.WriteString("### 回應時間分佈\n\n")
	sb.WriteString("| 場景 | 最小(ms) | P50(ms) | P95(ms) | P99(ms) | 最大(ms) | 平均(ms) |\n")
	sb.WriteString("|------|---------|---------|---------|---------|---------|----------|\n")

	for _, r := range results {
		if r.SimResult == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f |\n",
			r.Scenario.Name,
			r.SimResult.MinResponseTime,
			r.P50,
			r.P95,
			r.P99,
			r.SimResult.MaxResponseTime,
			r.SimResult.AvgResponseTime,
		))
	}

	sb.WriteString("\n### 吞吐量分析\n\n")
	sb.WriteString("| 場景 | 併發數 | 總請求 | 吞吐量(req/s) | 場景耗時 |\n")
	sb.WriteString("|------|--------|--------|--------------|----------|\n")

	for _, r := range results {
		if r.SimResult == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.0f | %v |\n",
			r.Scenario.Name,
			r.Scenario.Concurrency,
			r.SimResult.TotalRequests,
			r.Throughput,
			r.Duration.Round(time.Millisecond),
		))
	}

	sb.WriteString("\n### 效能趨勢觀察\n\n")

	// 找出最高吞吐量
	var maxThroughput float64
	var maxThroughputName string
	for _, r := range results {
		if r.Throughput > maxThroughput {
			maxThroughput = r.Throughput
			maxThroughputName = r.Scenario.Name
		}
	}

	sb.WriteString(fmt.Sprintf("- **最高吞吐量**: %s 場景達到 **%.0f req/s**\n", maxThroughputName, maxThroughput))
	sb.WriteString("- 隨著併發數增加，由於 Mutex 序列化搶票操作，回應時間會略微上升\n")
	sb.WriteString("- 但 Go 的 goroutine 調度效率確保了即使在極高併發下，系統依然穩定運作\n")
	sb.WriteString("- 由於直接呼叫 Service 層（無 HTTP 開銷），吞吐量數據反映的是純粹的併發處理能力\n\n")
	sb.WriteString("---\n\n")
}

// writeConcurrencyAdvantages 寫入 Go 併發優勢總結
func writeConcurrencyAdvantages(sb *strings.Builder) {
	sb.WriteString("## Go 併發優勢總結\n\n")

	sb.WriteString("### 1. goroutine 輕量級併發\n\n")
	sb.WriteString("- 每個 goroutine 初始僅佔用 **2-8 KB** 棧記憶體（對比 Java 線程的 1 MB）\n")
	sb.WriteString("- 本測試中最大同時啟動 **10,000 個 goroutine**，記憶體開銷極低\n")
	sb.WriteString("- Go runtime 的 M:N 調度器將 goroutine 高效分配到 OS 線程上\n\n")

	sb.WriteString("### 2. sync.Mutex 確保正確性\n\n")
	sb.WriteString("- 搶票的「讀庫存→判斷→扣減→下單」四步驟在同一把 Mutex 內完成\n")
	sb.WriteString("- 即使 10,000 個 goroutine 同時搶 100 張票，也**絕不超賣**\n")
	sb.WriteString("- 不使用 `defer Unlock()`，手動控制解鎖時機以減少鎖持有時間\n\n")

	sb.WriteString("### 3. sync/atomic 高效統計\n\n")
	sb.WriteString("- 計數器（嘗試次數、成功數、失敗數）使用 `atomic.AddInt64`\n")
	sb.WriteString("- 無需加鎖，硬體層級的原子操作，效能遠優於 Mutex\n")
	sb.WriteString("- 適用於「只需要累加/讀取」的場景\n\n")

	sb.WriteString("### 4. channel + WaitGroup 模式\n\n")
	sb.WriteString("- **fan-out**: 啟動 N 個 goroutine 併發搶票\n")
	sb.WriteString("- **fan-in**: 透過 buffered channel 收集所有結果\n")
	sb.WriteString("- WaitGroup 確保所有 goroutine 完成後才關閉 channel\n")
	sb.WriteString("- 這是 Go 併發程式設計的經典模式，簡潔且高效\n\n")

	sb.WriteString("### 5. 兩層鎖分離設計\n\n")
	sb.WriteString("- **Store 層**: `sync.RWMutex` 保護 map，允許多個讀取同時進行\n")
	sb.WriteString("- **Service 層**: `sync.Mutex` 保護搶票業務邏輯的原子性\n")
	sb.WriteString("- 兩層鎖各司其職，減少不必要的鎖競爭\n\n")
	sb.WriteString("---\n\n")
}

// writeConclusion 寫入結論
func writeConclusion(sb *strings.Builder, results []ScenarioResult) {
	sb.WriteString("## 結論\n\n")

	// 統計全域數據
	totalRequests := 0
	totalSuccess := 0
	totalFail := 0
	allPassed := true
	maxConcurrency := 0

	for _, r := range results {
		if r.SimResult != nil {
			totalRequests += r.SimResult.TotalRequests
			totalSuccess += r.SimResult.SuccessCount
			totalFail += r.SimResult.FailCount
		}
		if r.Oversold {
			allPassed = false
		}
		if r.Scenario.Concurrency > maxConcurrency {
			maxConcurrency = r.Scenario.Concurrency
		}
	}

	sb.WriteString("| 指標 | 數值 |\n")
	sb.WriteString("|------|------|\n")
	sb.WriteString(fmt.Sprintf("| 測試場景數 | %d |\n", len(results)))
	sb.WriteString(fmt.Sprintf("| 總請求數 | %d |\n", totalRequests))
	sb.WriteString(fmt.Sprintf("| 總成功數 | %d |\n", totalSuccess))
	sb.WriteString(fmt.Sprintf("| 總失敗數 | %d |\n", totalFail))
	sb.WriteString(fmt.Sprintf("| 最高併發數 | %d |\n", maxConcurrency))

	if allPassed {
		sb.WriteString("| 超賣事件 | **0（零超賣）** |\n")
	} else {
		sb.WriteString("| 超賣事件 | **存在超賣** |\n")
	}

	sb.WriteString("\n")
	if allPassed {
		sb.WriteString("本次壓力測試充分驗證了 Go 搶票模擬系統的**併發正確性**與**高效能**：\n\n")
		sb.WriteString(fmt.Sprintf("1. 在 %d 個測試場景中，最高併發 %d 用戶同時搶票，**零超賣事件**\n", len(results), maxConcurrency))
		sb.WriteString("2. `sync.Mutex` 確保搶票操作原子性，守恆等式在每個場景中嚴格成立\n")
		sb.WriteString("3. Go 的 goroutine 輕量化設計使得萬級併發毫無壓力\n")
		sb.WriteString("4. `sync/atomic` 在統計場景中提供了無鎖高效能方案\n")
		sb.WriteString("5. fan-out/fan-in 模式展現了 Go channel 的優雅併發協調能力\n")
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString(fmt.Sprintf("*報告由 `cmd/report/main.go` 自動生成 — %s*\n",
		time.Now().Format("2006-01-02 15:04:05")))
}
