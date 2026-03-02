# Go 搶票模擬系統 — 使用說明書

## 1. 系統簡介

這是一個用 Go 語言開發的搶票模擬後端系統，用於學習 Go 的高併發處理模式。系統模擬大量用戶同時搶購限量門票的場景，展示 goroutine、sync.Mutex、channel、WaitGroup、sync/atomic 等核心併發機制的實際應用。

## 2. 環境需求

- Go 1.22 以上
- 無需安裝資料庫（純記憶體儲存）

## 3. 啟動伺服器

```bash
cd go_api_01
go run main.go
```

啟動後會看到：
```
搶票模擬系統啟動中，監聽 :8080 ...
```

伺服器預設監聽 `http://localhost:8080`。

## 4. API 使用說明

### 4.1 建立活動

建立一場售票活動，指定名稱、描述、總票數與票價。

```bash
curl -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "name": "五月天演唱會",
    "description": "2026巡迴",
    "total_tickets": 100,
    "price": 2800
  }'
```

回應範例：
```json
{
  "code": 0,
  "message": "活動建立成功",
  "data": {
    "id": "evt-1",
    "name": "五月天演唱會",
    "description": "2026巡迴",
    "total_tickets": 100,
    "remaining": 100,
    "price": 2800,
    "created_at": "2026-03-02T13:31:40Z"
  }
}
```

### 4.2 查詢活動

查詢活動詳情，包含剩餘票數。

```bash
curl http://localhost:8080/api/events/evt-1
```

### 4.3 搶票

指定用戶 ID 和購買數量進行搶票。

```bash
curl -X POST http://localhost:8080/api/events/evt-1/grab \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-001",
    "quantity": 2
  }'
```

成功回應：
```json
{
  "code": 0,
  "message": "搶票成功",
  "data": {
    "user_id": "user-001",
    "success": true,
    "message": "搶票成功！訂單編號: ord-1",
    "response_time_ms": 0.05
  }
}
```

票數不足時回應（HTTP 409）：
```json
{
  "code": 1,
  "message": "票數不足，剩餘 0 張",
  "data": {
    "user_id": "user-002",
    "success": false,
    "message": "票數不足，剩餘 0 張"
  }
}
```

### 4.4 查詢訂單

查詢指定活動的所有成功訂單。

```bash
curl http://localhost:8080/api/events/evt-1/orders
```

回應範例：
```json
{
  "code": 0,
  "message": "查詢成功",
  "data": {
    "event_id": "evt-1",
    "total_count": 100,
    "orders": [
      {
        "id": "ord-1",
        "event_id": "evt-1",
        "user_id": "sim-user-42",
        "quantity": 1,
        "total_price": 2800,
        "created_at": "2026-03-02T13:33:08Z"
      }
    ]
  }
}
```

### 4.5 啟動搶票模擬

模擬大量用戶同時搶票。指定目標活動、併發數量和每人搶購張數。

```bash
curl -X POST http://localhost:8080/api/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-1",
    "concurrency": 500,
    "per_user": 1
  }'
```

回應會包含模擬統計摘要與每位用戶的搶票結果：
```json
{
  "code": 0,
  "message": "模擬完成",
  "data": {
    "event_id": "evt-1",
    "total_requests": 500,
    "success_count": 100,
    "fail_count": 400,
    "success_rate": 20,
    "avg_response_time_ms": 0.04,
    "min_response_time_ms": 0,
    "max_response_time_ms": 0.5,
    "results": [ ... ]
  }
}
```

### 4.6 查看全域統計

查看系統啟動以來的累計統計資訊。

```bash
curl http://localhost:8080/api/stats
```

回應範例：
```json
{
  "code": 0,
  "message": "查詢成功",
  "data": {
    "total_attempts": 501,
    "total_success": 100,
    "total_fail": 401,
    "success_rate": 19.96,
    "avg_response_time_ms": 0.001,
    "total_response_time_ms": 2.61
  }
}
```

## 5. 典型使用流程

### 快速體驗（3 步驟）

```bash
# 1. 啟動伺服器
go run main.go

# 2. 建立 100 張票的活動
curl -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d '{"name":"測試活動","total_tickets":100,"price":500}'

# 3. 模擬 500 人同時搶票，觀察成功率約 20%
curl -X POST http://localhost:8080/api/simulate \
  -H "Content-Type: application/json" \
  -d '{"event_id":"evt-1","concurrency":500,"per_user":1}'
```

### 進階測試：觀察不同併發比例

| 場景 | 總票數 | 併發數 | 每人搶 | 預期成功率 |
|------|--------|--------|--------|-----------|
| 低競爭 | 100 | 50 | 1 | ~100% |
| 中競爭 | 100 | 200 | 1 | ~50% |
| 高競爭 | 100 | 500 | 1 | ~20% |
| 超高競爭 | 100 | 1000 | 1 | ~10% |
| 大量搶購 | 100 | 200 | 2 | ~25% |

> 注意：每次模擬前需建立新活動，因為票數用完後無法重複搶票。

## 6. 統一回應格式

所有 API 回應皆遵循以下格式：

```json
{
  "code": 0,        // 0=成功，非0=錯誤
  "message": "...", // 說明訊息
  "data": { ... }   // 回應資料（錯誤時可能為空）
}
```

## 7. 錯誤碼對照

| HTTP 狀態碼 | 說明 |
|------------|------|
| 200 | 操作成功 |
| 201 | 建立成功 |
| 400 | 請求參數錯誤 |
| 404 | 活動不存在 |
| 409 | 搶票失敗（票數不足） |

## 8. 注意事項

- 系統使用**記憶體儲存**，重啟伺服器後所有資料會清除。
- 活動 ID 從 `evt-1` 開始自動遞增，訂單 ID 從 `ord-1` 開始。
- 模擬器為**同步阻塞**呼叫，500 人模擬通常在 1 秒內完成。
- 併發數建議不超過 10000，避免記憶體佔用過大。
