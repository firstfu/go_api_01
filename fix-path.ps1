# 修復被 setx 截斷的 User PATH
# 只保留使用者獨有的路徑，移除與 System PATH 重複的項目，並加入 GnuWin32

$machinePath = [Environment]::GetEnvironmentVariable('PATH', 'Machine') -split ';' | Where-Object { $_ -ne '' }
$currentUser = [Environment]::GetEnvironmentVariable('PATH', 'User') -split ';' | Where-Object { $_ -ne '' }

# 找出 User PATH 中不在 Machine PATH 裡的項目
$uniqueUser = $currentUser | Where-Object { $_ -notin $machinePath }

# 加入 GnuWin32（如果還沒有的話）
$gnuWin32 = 'C:\Program Files (x86)\GnuWin32\bin'
if ($gnuWin32 -notin $uniqueUser) {
    $uniqueUser += $gnuWin32
}

$newUserPath = $uniqueUser -join ';'

Write-Host "=== 修復後的 User PATH ===" -ForegroundColor Green
$uniqueUser | ForEach-Object { Write-Host "  $_" }
Write-Host ""
Write-Host "共 $($uniqueUser.Count) 項, 長度 $($newUserPath.Length) 字元" -ForegroundColor Cyan
Write-Host ""

# 確認後寫入
$confirm = Read-Host "確定要寫入嗎？(y/N)"
if ($confirm -eq 'y') {
    [Environment]::SetEnvironmentVariable('PATH', $newUserPath, 'User')
    Write-Host "User PATH 已修復！請重新開啟終端機。" -ForegroundColor Green
} else {
    Write-Host "取消操作。" -ForegroundColor Yellow
}
