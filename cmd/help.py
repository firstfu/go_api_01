# cmd/help.py
# 顯示 Makefile 所有可用指令的說明
# 獨立為 Python 檔案以避免 Windows 終端機 codepage 亂碼

import sys
import io

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8")

print("""Go 搶票模擬系統 — 可用指令
────────────────────────────────────────

  開發：
    make run             啟動伺服器
    make build           編譯執行檔
    make test            執行測試
    make tidy            整理依賴

  API 操作（需先啟動伺服器）：
    make create-event    建立活動（100張票）
    make query-event     查詢活動        ID=evt-2
    make grab            單人搶票        ID=evt-1 USER=user-001 QTY=1
    make orders          查詢訂單        ID=evt-1
    make stats           查看全域統計

  模擬：
    make sim             模擬搶票        ID=evt-1 N=500 QTY=1
    make demo            完整流程展示（建立→模擬→統計）

  瀏覽器開啟：
    make open-event      開啟活動資訊    ID=evt-1
    make open-orders     開啟訂單列表    ID=evt-1
    make open-stats      開啟全域統計

  範例：
    make sim N=1000              1000人搶票
    make sim ID=evt-2 N=200      指定活動、200人
    make grab ID=evt-3 QTY=2     搶 evt-3 的 2 張票""")
