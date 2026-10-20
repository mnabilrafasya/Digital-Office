<#
.SYNOPSIS
  Smoke test E2E Sprint 2 digioffice (skenario Lampiran B) terhadap API lokal.

.DESCRIPTION
  Menjalankan alur surat lengkap: draf, ETag, unggah paralel ke MinIO, submit bersamaan,
  review/return/approve, tanda tangan, cek inbox di database, rate limit login, dan logout.
  Password user dummy diminta dengan Read-Host -AsSecureString dan tidak pernah dicetak.
  Token, password, dan presigned URL tidak pernah ditampilkan.

  Prasyarat: API berjalan (go run ./cmd/api), database sudah di-migrate dan di-seed (seed all).
  Jalankan dari folder backend:
    powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1

.PARAMETER BaseUrl
  Alamat API. Default http://localhost:8080
.PARAMETER SeedPassword
  Password user dummy (SecureString). Bila kosong, diminta lewat Read-Host.
.PARAMETER DbName
  Database untuk cek inbox (langkah 18). Default: DB_NAME di environment/.env, lalu "digioffice".
#>
param(
    [string]$BaseUrl = 'http://localhost:8080',
    [System.Security.SecureString]$SeedPassword,
    [string]$DbName
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http
Set-Location (Split-Path -Parent $PSScriptRoot)

if (-not $SeedPassword) { $SeedPassword = Read-Host -AsSecureString 'Password user dummy (SEED_PASSWORD)' }
$bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($SeedPassword)
try { $script:pw = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr) } finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr) }

function Get-EnvValue([string]$key) {
    if ([Environment]::GetEnvironmentVariable($key)) { return [Environment]::GetEnvironmentVariable($key) }
    if (Test-Path .env) {
        $m = [regex]::Match([IO.File]::ReadAllText("$PWD\.env"), "(?m)^$key=(.*?)\r?$")
        if ($m.Success) { return $m.Groups[1].Value.Trim().Trim('"') }
    }
    return $null
}
if (-not $DbName) { $DbName = Get-EnvValue 'DB_NAME' }
if (-not $DbName) { $DbName = 'digioffice' }

$client = New-Object System.Net.Http.HttpClient
$client.Timeout = [TimeSpan]::FromMinutes(5)

# ---------- helper HTTP ----------
function New-Request([string]$method, [string]$url, [string]$token, $body, [hashtable]$headers) {
    $req = New-Object System.Net.Http.HttpRequestMessage((New-Object System.Net.Http.HttpMethod($method)), $url)
    if ($token) { [void]$req.Headers.TryAddWithoutValidation('Authorization', "Bearer $token") }
    if ($headers) { foreach ($k in $headers.Keys) { [void]$req.Headers.TryAddWithoutValidation($k, $headers[$k]) } }
    if ($null -ne $body) {
        $json = if ($body -is [string]) { $body } else { $body | ConvertTo-Json -Depth 10 -Compress }
        $req.Content = New-Object System.Net.Http.StringContent($json, [Text.Encoding]::UTF8, 'application/json')
    }
    return $req
}
function Read-Response($res) {
    $text = $res.Content.ReadAsStringAsync().Result
    $json = $null
    if ($text) { try { $json = $text | ConvertFrom-Json } catch {} }
    $etag = $null
    if ($res.Headers.ETag) { $etag = $res.Headers.ETag.ToString() }
    [pscustomobject]@{ Status = [int]$res.StatusCode; Json = $json; ETag = $etag; Res = $res }
}
function Call([string]$method, [string]$path, [string]$token = $null, $body = $null, [hashtable]$headers = $null) {
    Read-Response ($client.SendAsync((New-Request $method "$BaseUrl$path" $token $body $headers)).Result)
}
function Send-Concurrent([object[]]$requests) {
    $tasks = @($requests | ForEach-Object { $client.SendAsync($_) })
    [Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]$tasks)
    return @($tasks | ForEach-Object { Read-Response $_.Result })
}
function Login([string]$nik) {
    $r = Call 'POST' '/api/v1/auth/login' $null @{ employeeNumber = $nik; password = $script:pw }
    if ($r.Status -ne 200) { throw "Login $nik gagal: HTTP $($r.Status)" }
    return $r.Json.data
}
function ErrText($r) { if ($r.Json -and $r.Json.error) { "$($r.Status) $($r.Json.error.code)" } else { "$($r.Status)" } }

# Unggah langsung ke MinIO dengan presigned POST: semua field apa adanya, file terakhir.
# Field "Content-Type" sudah ada di fields (kosong); nilainya diganti tipe file, bukan ditambah field baru.
function New-UploadRequest($ticket, [byte[]]$bytes, [string]$fileName, [string]$contentType) {
    $form = New-Object System.Net.Http.MultipartFormDataContent
    foreach ($p in $ticket.fields.PSObject.Properties) {
        $value = if ($p.Name -eq 'Content-Type') { $contentType } else { [string]$p.Value }
        $sc = New-Object System.Net.Http.StringContent($value)
        $sc.Headers.ContentType = $null
        $form.Add($sc, $p.Name)
    }
    $file = New-Object System.Net.Http.ByteArrayContent(, $bytes)
    $form.Add($file, 'file', $fileName)
    $req = New-Object System.Net.Http.HttpRequestMessage([System.Net.Http.HttpMethod]::Post, $ticket.uploadUrl)
    $req.Content = $form
    return $req
}
function New-Bytes([int]$size) { $b = New-Object byte[] $size; (New-Object Random).NextBytes($b); return , $b }

# ---------- hasil ----------
$rows = New-Object System.Collections.Generic.List[object]
function Step($no, [string]$name, [string]$expect, [string]$actual, [bool]$ok) {
    $rows.Add([pscustomobject]@{ No = $no; Langkah = $name; Harapan = $expect; Hasil = $actual; Status = $(if ($ok) { 'LULUS' } else { 'GAGAL' }) })
}

try {
    # ---------- 1. Login + alias ----------
    $andi = Login '2015047'; $tA = $andi.accessToken
    $aliases = (Call 'GET' '/api/v1/recipients/aliases' $tA).Json.data
    $vp = $aliases | Where-Object { $_.code -eq 'VP' }
    Step 1 'Andi login, GET /recipients/aliases' 'VP memberCount 2' "VP memberCount $($vp.memberCount)" ($vp.memberCount -eq 2)

    # Data referensi untuk draf
    $lk = { param($cat, $code) ((Call 'GET' "/api/v1/lookups?category=$cat" $tA).Json.data | Where-Object { $_.code -eq $code }).id }
    $letterType = & $lk 'LETTER_TYPE' 'NOTA_DINAS'
    $classif = & $lk 'CLASSIFICATION' 'TI.01'
    $conf = & $lk 'CONFIDENTIALITY' 'BIASA'
    $urg = & $lk 'URGENCY' 'SEGERA'
    $cand = { param($type, $q) ((Call 'GET' "/api/v1/users/candidates?type=$type&q=$q" $tA).Json.data)[0].id }
    $fajarId = & $cand 'REVIEWER' 'Fajar'
    $dewiId = & $cand 'REVIEWER' 'Dewi'
    $agusId = & $cand 'APPROVER' 'Agus'
    $unitId = { param($code) ((Call 'GET' '/api/v1/units' $tA).Json.data | Where-Object { $_.code -eq $code }).id }
    $sdm = & $unitId 'SDM'; $ak = & $unitId 'AK'

    # ---------- 2. Draf + PUT lengkap ----------
    $created = Call 'POST' '/api/v1/letters' $tA @{}
    $letter = $created.Json.data.id
    $etag0 = $created.ETag
    $draft = @{
        letterTypeId = $letterType; classificationId = $classif; confidentialityId = $conf; urgencyId = $urg
        title = 'Undangan Rapat Koordinasi Sistem Informasi (smoke test)'; summary = 'Ringkasan uji'; specialNote = 'Catatan uji'
        reviewerIds = @($fajarId, $dewiId); approverId = $agusId
        recipients = @{ to = @(@{ unitId = $sdm }, @{ aliasLevelId = $vp.levelId }); cc = @(@{ unitId = $ak }) }
    }
    $put = Call 'PUT' "/api/v1/letters/$letter" $tA $draft @{ 'If-Match' = $etag0 }
    Step 2 'Buat draf lalu PUT lengkap' '201 lalu 200, ETag berubah' "$($created.Status) lalu $($put.Status), ETag berubah: $($put.ETag -ne $etag0)" ($created.Status -eq 201 -and $put.Status -eq 200 -and $put.ETag -and $put.ETag -ne $etag0)
    $etag1 = $put.ETag

    # ---------- 3-5. Tabrakan dan validasi ----------
    $r = Call 'PUT' "/api/v1/letters/$letter" $tA $draft @{ 'If-Match' = $etag0 }
    Step 3 'PUT dengan ETag lama' '412' (ErrText $r) ($r.Status -eq 412 -and $r.Json.error.code -eq 'VERSION_CONFLICT')
    $r = Call 'PUT' "/api/v1/letters/$letter" $tA $draft
    Step 4 'PUT tanpa If-Match' '428' (ErrText $r) ($r.Status -eq 428)
    $self = $draft.Clone(); $self.approverId = $andi.user.id
    $r = Call 'PUT' "/api/v1/letters/$letter" $tA $self @{ 'If-Match' = $etag1 }
    Step 5 'Approver = Andi sendiri' '400' (ErrText $r) ($r.Status -eq 400)

    # ---------- 6. Dua lampiran paralel + satu versi ----------
    $t1 = (Call 'POST' "/api/v1/letters/$letter/attachments/upload-url" $tA @{ fileName = 'Lampiran A.pdf'; size = 2048 }).Json.data
    $t2 = (Call 'POST' "/api/v1/letters/$letter/attachments/upload-url" $tA @{ fileName = 'Lampiran B.xlsx'; size = 4096 }).Json.data
    $up = Send-Concurrent @((New-UploadRequest $t1 (New-Bytes 2048) 'Lampiran A.pdf' 'application/pdf'), (New-UploadRequest $t2 (New-Bytes 4096) 'Lampiran B.xlsx' 'application/vnd.ms-excel'))
    $done = Send-Concurrent @((New-Request 'POST' "$BaseUrl/api/v1/letters/$letter/attachments" $tA @{ uploadToken = $t1.uploadToken }), (New-Request 'POST' "$BaseUrl/api/v1/letters/$letter/attachments" $tA @{ uploadToken = $t2.uploadToken }))
    $tv = (Call 'POST' "/api/v1/letters/$letter/versions/upload-url" $tA @{ fileName = 'Nota Dinas.docx'; size = 1024 }).Json.data
    $upv = Send-Concurrent @((New-UploadRequest $tv (New-Bytes 1024) 'Nota Dinas.docx' 'application/octet-stream'))
    $ver = Call 'POST' "/api/v1/letters/$letter/versions" $tA @{ uploadToken = $tv.uploadToken }
    $detail = (Call 'GET' "/api/v1/letters/$letter" $tA).Json.data
    $minio = (@($up + $upv) | ForEach-Object { $_.Status }) -join '/'
    Step 6 '2 lampiran paralel + 1 versi' 'keduanya tercatat, versi 1' "MinIO $minio; konfirmasi $($done[0].Status)/$($done[1].Status)/$($ver.Status); lampiran $(@($detail.attachments).Count), versi $($detail.currentVersion)" (@($detail.attachments).Count -eq 2 -and $detail.currentVersion -eq 1 -and $ver.Json.data.versionNumber -eq 1)

    # ---------- 7. Perusahaan lain ----------
    $tL = (Login '2019203').accessToken
    $r = Call 'GET' "/api/v1/letters/$letter" $tL
    Step 7 'Lestari (PKG) membuka surat Andi' '404' (ErrText $r) ($r.Status -eq 404)

    # ---------- 8. Submit bersamaan ----------
    $sub = Send-Concurrent @((New-Request 'POST' "$BaseUrl/api/v1/letters/$letter/submit" $tA $null), (New-Request 'POST' "$BaseUrl/api/v1/letters/$letter/submit" $tA $null))
    $codes = @($sub | ForEach-Object { $_.Status } | Sort-Object)
    Step 8 'Submit 2 request bersamaan' 'satu 200, satu 409' ($codes -join ' dan ') (($codes -join ',') -eq '200,409')

    # ---------- 9-12. Review, return, kirim ulang ----------
    $tF = (Login '2012060').accessToken; $tD = (Login '2003035').accessToken; $tG = (Login '1995021').accessToken
    $r = Call 'POST' "/api/v1/letters/$letter/approve" $tF @{ note = 'Sudah sesuai' }
    Step 9 'Fajar approve' 'assignee Dewi' "$($r.Status) $($r.Json.data.status), assignee $($r.Json.data.currentAssignee.name)" ($r.Status -eq 200 -and $r.Json.data.currentAssignee.id -eq $dewiId)
    $r = Call 'POST' "/api/v1/letters/$letter/return" $tD @{ note = 'Mohon perbaiki perihal' }
    Step 10 'Dewi return (catatan)' 'RETURNED_TO_DRAFTER, assignee Andi' "$($r.Status) $($r.Json.data.status), assignee $($r.Json.data.currentAssignee.name)" ($r.Status -eq 200 -and $r.Json.data.status -eq 'RETURNED_TO_DRAFTER' -and $r.Json.data.currentAssignee.id -eq $andi.user.id)
    $r = Call 'POST' "/api/v1/letters/$letter/submit" $tA
    Step 11 'Andi submit ulang' 'assignee Fajar (reviewer 1)' "$($r.Status) $($r.Json.data.status), assignee $($r.Json.data.currentAssignee.name)" ($r.Status -eq 200 -and $r.Json.data.currentAssignee.id -eq $fajarId)
    $r1 = Call 'POST' "/api/v1/letters/$letter/approve" $tF
    $r2 = Call 'POST' "/api/v1/letters/$letter/approve" $tD
    Step 12 'Fajar approve, Dewi approve' 'IN_APPROVAL, assignee Agus' "$($r1.Status)/$($r2.Status) $($r2.Json.data.status), assignee $($r2.Json.data.currentAssignee.name)" ($r2.Status -eq 200 -and $r2.Json.data.status -eq 'IN_APPROVAL' -and $r2.Json.data.currentAssignee.id -eq $agusId)

    # ---------- 13-15. Approver mengembalikan ----------
    $r = Call 'POST' "/api/v1/letters/$letter/return" $tG @{ note = '' }
    Step 13 'Agus return tanpa catatan' '400' (ErrText $r) ($r.Status -eq 400)
    $r = Call 'POST' "/api/v1/letters/$letter/return" $tG @{ note = 'Lampiran anggaran belum lengkap' }
    Step 14 'Agus return (catatan)' 'RETURNED_TO_REVIEWER, assignee Dewi' "$($r.Status) $($r.Json.data.status), assignee $($r.Json.data.currentAssignee.name)" ($r.Status -eq 200 -and $r.Json.data.status -eq 'RETURNED_TO_REVIEWER' -and $r.Json.data.currentAssignee.id -eq $dewiId)
    $r = Call 'POST' "/api/v1/letters/$letter/approve" $tD
    Step 15 'Dewi approve' 'IN_APPROVAL, assignee Agus langsung' "$($r.Status) $($r.Json.data.status), assignee $($r.Json.data.currentAssignee.name)" ($r.Status -eq 200 -and $r.Json.data.status -eq 'IN_APPROVAL' -and $r.Json.data.currentAssignee.id -eq $agusId)

    # ---------- 16-17. Tanda tangan ----------
    $r = Call 'POST' "/api/v1/letters/$letter/sign" $tF
    Step 16 'Fajar mencoba sign' '409 (peserta, bukan assignee)' (ErrText $r) ($r.Status -eq 409 -and $r.Json.error.code -eq 'INVALID_STATE')
    $r = Call 'POST' "/api/v1/letters/$letter/sign" $tG
    $roman = @('I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX', 'X', 'XI', 'XII')
    $jkt = [TimeZoneInfo]::ConvertTimeBySystemTimeZoneId([DateTime]::UtcNow, 'SE Asia Standard Time')
    $pattern = '^\d{3,}/PA/PSP/' + $roman[$jkt.Month - 1] + '/' + $jkt.Year + '$'
    $num = $r.Json.data.letterNumber
    Step 17 'Agus sign' 'SENT, nomor NNN/PA/PSP/{bulan}/{tahun}' "$($r.Status) $($r.Json.data.status), nomor $num" ($r.Status -eq 200 -and $r.Json.data.status -eq 'SENT' -and $num -match $pattern)

    # ---------- 18. Inbox di database ----------
    $dbPw = Get-EnvValue 'DB_PASSWORD'
    $q = "SET NOCOUNT ON; SELECT p.name + '|' + i.recipient_type FROM inbox_items i JOIN letters l ON l.id = i.letter_id JOIN users u ON u.id = i.user_id JOIN positions p ON p.id = u.position_id WHERE l.uuid = '$letter' ORDER BY p.name"
    $inbox = @(docker exec digioffice-sqlserver /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P $dbPw -C -d $DbName -h -1 -W -Q $q | Where-Object { $_ })
    $dbPw = $null
    $expected = @('VP Akuntansi dan Keuangan|TO', 'VP Sumber Daya Manusia|TO')
    Step 18 'Cek DB inbox_items' 'VP SDM 1x TO, VP AK 1x TO' ($inbox -join '; ') (($inbox -join ',') -eq ($expected -join ','))

    # ---------- 19. Riwayat ----------
    $hist = (Call 'GET' "/api/v1/letters/$letter/history" $tA).Json.data
    $flow = @($hist | Where-Object { $_.action -ne 'READ' } | ForEach-Object { $_.action })
    $want = @('SAVE', 'SUBMIT', 'APPROVE', 'RETURN', 'SUBMIT', 'APPROVE', 'APPROVE', 'RETURN', 'APPROVE', 'SIGN')
    $reads = @($hist | Where-Object { $_.action -eq 'READ' }).Count
    Step 19 'Riwayat surat' ($want -join '>') "$($flow -join '>') (+$reads READ)" (($flow -join ',') -eq ($want -join ','))

    # ---------- 20. Rate limit login ----------
    $nik = '9' + (Get-Random -Minimum 100000 -Maximum 999999)
    $codes = @()
    $retry = $null
    for ($i = 1; $i -le 6; $i++) {
        $r = Call 'POST' '/api/v1/auth/login' $null @{ employeeNumber = $nik; password = 'salah-sekali' }
        $codes += $r.Status
        if ($i -eq 6 -and $r.Res.Headers.RetryAfter) { $retry = $r.Res.Headers.RetryAfter.Delta.TotalSeconds }
    }
    Step 20 '6x login salah untuk NIK yang sama' 'ke-6: 429 + Retry-After' "$($codes -join ','), Retry-After $retry detik" ($codes[5] -eq 429 -and $retry -gt 0 -and ($codes[0..4] | Where-Object { $_ -ne 401 }).Count -eq 0)

    # ---------- 21. Logout ----------
    $out = Call 'POST' '/api/v1/auth/logout' $tA
    $r = Call 'GET' '/api/v1/auth/me' $tA
    Step 21 'Andi logout, pakai token lama' '401' "logout $($out.Status), token lama $(ErrText $r)" ($out.Status -eq 204 -and $r.Status -eq 401)
}
catch {
    Step 'X' 'Skenario berhenti karena error' '-' $_.Exception.Message $false
}
finally {
    $script:pw = $null
}

$rows | Format-Table -AutoSize -Wrap | Out-String -Width 220
$pass = @($rows | Where-Object { $_.Status -eq 'LULUS' }).Count
Write-Output ("Hasil: {0} LULUS dari {1} langkah" -f $pass, $rows.Count)
if ($pass -ne $rows.Count) { exit 1 }
