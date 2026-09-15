# MeowHome 后端冒烟测试
#
# 用法（本机 PowerShell 执行策略为 Restricted，需显式 Bypass）：
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\smoke.ps1
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\smoke.ps1 -BaseUrl http://127.0.0.1:18080
#
# 注意：本文件必须保存为「带 BOM 的 UTF-8」，否则 Windows PowerShell 5.1
# 会按 GBK 解析，中文字符串会破坏语法。
#
# 覆盖：健康检查、注册/登录/刷新/注销、/me、家庭、猫咪、
#       认证负向用例（篡改/伪造/过期令牌）、跨家庭隔离。

[CmdletBinding()]
param(
    [string]$BaseUrl = 'http://127.0.0.1:8080'
)

$ErrorActionPreference = 'Stop'
$api = "$BaseUrl/api/v1"
$script:okCount = 0
$script:failCount = 0

function Invoke-Api {
    param([string]$Method, [string]$Path, $Body, [string]$Token)
    $headers = @{}
    if ($Token) { $headers['Authorization'] = "Bearer $Token" }
    $params = @{ Method = $Method; Uri = "$api$Path"; Headers = $headers; ErrorAction = 'Stop' }
    if ($null -ne $Body) {
        # 必须显式送 UTF-8 字节：Windows PowerShell 5.1 的 Invoke-RestMethod
        # 对字符串 body 默认按 Latin-1/ASCII 编码，中文会被写成 "?"(0x3F)。
        $json = $Body | ConvertTo-Json -Compress
        $params['Body'] = [System.Text.Encoding]::UTF8.GetBytes($json)
        $params['ContentType'] = 'application/json; charset=utf-8'
    }
    try {
        return @{ status = 200; body = (Invoke-RestMethod @params) }
    } catch {
        $resp = $_.Exception.Response
        $code = 0; $text = ''
        if ($resp) {
            $code = [int]$resp.StatusCode
            try { $text = (New-Object System.IO.StreamReader($resp.GetResponseStream())).ReadToEnd() } catch {}
        }
        return @{ status = $code; body = $text }
    }
}

function Assert-Equal {
    param([string]$Label, $Actual, $Expected)
    if ("$Actual" -eq "$Expected") {
        Write-Host ("  [PASS] {0,-34} = {1}" -f $Label, $Actual) -ForegroundColor Green
        $script:okCount++
    } else {
        Write-Host ("  [FAIL] {0,-34} = {1}  (期望 {2})" -f $Label, $Actual, $Expected) -ForegroundColor Red
        $script:failCount++
    }
}

Write-Host "`n=== 健康检查 ===" -ForegroundColor Cyan
$live  = Invoke-RestMethod "$BaseUrl/health/live"
$ready = Invoke-RestMethod "$BaseUrl/health/ready"
Assert-Equal 'health/live status'  $live.data.status  'up'
Assert-Equal 'health/ready status' $ready.data.status 'up'

$email = "smoke-$([guid]::NewGuid().ToString('N').Substring(0,8))@test.com"
$pass  = 'password123'

Write-Host "`n=== 注册与令牌 ===" -ForegroundColor Cyan
$reg = Invoke-Api 'POST' '/auth/register' @{ email = $email; password = $pass; user_name = 'SmokeUser' } $null
Assert-Equal 'POST /auth/register' $reg.status 200
$token   = $reg.body.data.access_token
$refresh = $reg.body.data.refresh_token
Assert-Equal 'access_token 段数' ($token -split '\.').Count 2
Assert-Equal 'expires_in(秒)'    $reg.body.data.expires_in   900

Write-Host "`n=== 当前用户与家庭 ===" -ForegroundColor Cyan
$me = Invoke-Api 'GET' '/me' $null $token
Assert-Equal 'GET /me' $me.status 200
Assert-Equal 'me.user_name' $me.body.data.user_name 'SmokeUser'

$fam = Invoke-Api 'POST' '/families' @{ name = '冒烟猫宅' } $token
Assert-Equal 'POST /families' $fam.status 200
$familyId = $fam.body.data.id

$me2 = Invoke-Api 'GET' '/me' $null $token
Assert-Equal 'me.family_id 已注入' $me2.body.data.family_id $familyId
Assert-Equal 'me.role'             $me2.body.data.role      'owner'

$famList = Invoke-Api 'GET' '/families' $null $token
Assert-Equal 'GET /families 数量' $famList.body.data.Count 1

Write-Host "`n=== 猫咪 ===" -ForegroundColor Cyan
$cat = Invoke-Api 'POST' "/families/$familyId/cats" @{ name = '小白'; breed = '中华田园猫'; gender = 'female' } $token
Assert-Equal 'POST cats' $cat.status 200
$cats = Invoke-Api 'GET' "/families/$familyId/cats" $null $token
Assert-Equal 'GET cats 数量' $cats.body.data.Count 1
Assert-Equal 'cat.name'      $cats.body.data[0].name '小白'

Write-Host "`n=== 令牌刷新与轮换 ===" -ForegroundColor Cyan
$rf = Invoke-Api 'POST' '/auth/refresh' @{ refresh_token = $refresh } $null
Assert-Equal 'POST /auth/refresh' $rf.status 200
Assert-Equal '旧 refresh 复用被拒' (Invoke-Api 'POST' '/auth/refresh' @{ refresh_token = $refresh } $null).status 401

Write-Host "`n=== 认证负向用例 ===" -ForegroundColor Cyan
Assert-Equal '无 token'      (Invoke-Api 'GET' '/me' $null $null).status 401
Assert-Equal '乱码 token'    (Invoke-Api 'GET' '/me' $null 'garbage').status 401
Assert-Equal '篡改签名'      (Invoke-Api 'GET' '/me' $null ($token.Substring(0, $token.Length - 4) + 'ffff')).status 401
Assert-Equal '伪造 uid 段'   (Invoke-Api 'GET' '/me' $null '01JABCDEFGHJKMNPQRSTVWXYZ00.abcdef').status 401
Assert-Equal '错误密码'      (Invoke-Api 'POST' '/auth/login' @{ email = $email; password = 'wrong-pass-123' } $null).status 401
Assert-Equal '重复注册'      (Invoke-Api 'POST' '/auth/register' @{ email = $email; password = $pass; user_name = 'dup' } $null).status 409
Assert-Equal '正确密码登录'  (Invoke-Api 'POST' '/auth/login' @{ email = $email; password = $pass } $null).status 200

Write-Host "`n=== 跨家庭隔离 ===" -ForegroundColor Cyan
$other = "smoke-iso-$([guid]::NewGuid().ToString('N').Substring(0,8))@test.com"
$t2 = (Invoke-Api 'POST' '/auth/register' @{ email = $other; password = $pass; user_name = 'Intruder' } $null).body.data.access_token
Assert-Equal '读他人家庭'     (Invoke-Api 'GET' "/families/$familyId" $null $t2).status 403
Assert-Equal '读他人猫咪列表' (Invoke-Api 'GET' "/families/$familyId/cats" $null $t2).status 403
Assert-Equal '他人家庭列表为空' (Invoke-Api 'GET' '/families' $null $t2).body.data.Count 0

Write-Host "`n================================" -ForegroundColor Cyan
$color = if ($script:failCount -eq 0) { 'Green' } else { 'Red' }
Write-Host ("通过 {0} / 失败 {1}" -f $script:okCount, $script:failCount) -ForegroundColor $color
exit $script:failCount
