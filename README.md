# Go 搶票模擬系統

一個用 Go 開發的高併發搶票模擬後端系統，用於學習 goroutine、Mutex、channel、WaitGroup、atomic 等核心併發機制。

## 技術棧

- **Go 1.22+** / **Gin** 框架
- 純記憶體儲存（無資料庫依賴）
- sync.Mutex（搶票原子性）、sync.RWMutex（map 保護）、sync/atomic（統計計數器）
- goroutine + WaitGroup + buffered channel（fan-out/fan-in 模擬器）

## 專案結構

```
go_api_01/
├── main.go                 # 進入點，組裝依賴並啟動伺服器
├── router/router.go        # 路由設定 + CORS 中間件
├── handler/                # HTTP 處理器層
│   ├── event.go            #   活動 CRUD
│   ├── ticket.go           #   搶票 + 訂單查詢
│   └── simulate.go         #   模擬啟動 + 統計查詢
├── service/                # 業務邏輯層
│   ├── event.go            #   活動管理
│   ├── ticket.go           #   搶票核心（Mutex 鎖）
│   └── simulator.go        #   併發模擬器（goroutine + channel）
├── model/model.go          # 資料結構定義
├── store/store.go          # 線程安全的記憶體存儲
└── docs/                   # 文件
    ├── prd.md              #   產品需求文件
    ├── todo.md             #   開發待辦清單
    └── usage.md            #   使用說明書
```

## 快速開始

```bash
# 安裝依賴
go mod tidy

# 啟動伺服器（監聽 :8080）
go run main.go
```

## API 端點

| Method | Path | 說明 |
|--------|------|------|
| POST | `/api/events` | 建立活動 |
| GET | `/api/events/:id` | 查詢活動 |
| POST | `/api/events/:id/grab` | 搶票 |
| GET | `/api/events/:id/orders` | 查詢訂單 |
| POST | `/api/simulate` | 啟動搶票模擬 |
| GET | `/api/stats` | 查看全域統計 |

## 使用範例

```bash
# 1. 建立 100 張票的活動
curl -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d '{"name":"演唱會","total_tickets":100,"price":2800}'

# 2. 手動搶票
curl -X POST http://localhost:8080/api/events/evt-1/grab \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-001","quantity":2}'

# 3. 模擬 500 人搶 100 張票（成功率約 20%）
curl -X POST http://localhost:8080/api/simulate \
  -H "Content-Type: application/json" \
  -d '{"event_id":"evt-1","concurrency":500,"per_user":1}'

# 4. 查看統計
curl http://localhost:8080/api/stats
```

## 核心併發設計

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Handler    │────▶│     Service      │────▶│      Store       │
│  (HTTP 處理) │     │  (業務邏輯)       │     │  (記憶體存儲)     │
└─────────────┘     └──────────────────┘     └──────────────────┘
                     │                        │
                     │ ticket.mu (Mutex)      │ store.mu (RWMutex)
                     │ 搶票業務原子性          │ map 結構安全
                     │                        │
                     │                        │ atomic 操作
                     │                        │ 統計計數器
                     └────────────────────────┘

模擬器 fan-out/fan-in：
  main ──┬── goroutine 1 ──┐
         ├── goroutine 2 ──┤
         ├── goroutine 3 ──┼──▶ buffered channel ──▶ 收集結果
         ├── ...           │
         └── goroutine N ──┘
```

## 詳細文件

- [產品需求文件](docs/prd.md)
- [使用說明書](docs/usage.md)
- [開發待辦清單](docs/todo.md)
