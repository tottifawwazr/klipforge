[CmdletBinding()]
param([int]$WebPort=3002,[int]$ApiPort=8082,[int]$PostgresPort=54329,[int]$RedisPort=16379,[switch]$RestartApps)
$ErrorActionPreference='Stop'
$projectRoot=Split-Path -Parent $PSScriptRoot
$cacheRoot=Join-Path $projectRoot '.cache'
if($RestartApps){
 foreach($applicationPort in @($WebPort,$ApiPort)){
  foreach($listener in @(Get-NetTCPConnection -LocalPort $applicationPort -State Listen -ErrorAction SilentlyContinue)){
   $applicationProcess=Get-CimInstance Win32_Process -Filter "ProcessId=$($listener.OwningProcess)"
   if(-not $applicationProcess.CommandLine -or $applicationProcess.CommandLine.IndexOf($projectRoot,[StringComparison]::OrdinalIgnoreCase) -lt 0){throw "Port $applicationPort is owned by a process outside this checkout. It was not stopped."}
   $applicationParent=Get-CimInstance Win32_Process -Filter "ProcessId=$($applicationProcess.ParentProcessId)"
   Stop-Process -Id $applicationProcess.ProcessId -Force
   if($applicationParent.CommandLine -and ($applicationParent.CommandLine -like '*scripts\start-preview.mjs*' -or $applicationParent.CommandLine -like '*node_modules\next\dist\bin\next*') -and $applicationParent.CommandLine.IndexOf($projectRoot,[StringComparison]::OrdinalIgnoreCase) -ge 0){Stop-Process -Id $applicationParent.ProcessId -Force -ErrorAction SilentlyContinue}
  }
 }
}

$goBinary=Join-Path $cacheRoot 'tools\go\bin\go.exe'
$pgRoot=Join-Path $cacheRoot 'tools\pgsql\bin'
$redisRoot=Join-Path $cacheRoot 'tools\redis\Redis-7.4.11-Windows-x64-msys2'
foreach($binary in @($goBinary,(Join-Path $pgRoot 'postgres.exe'),(Join-Path $redisRoot 'redis-server.exe'))){
 if(-not(Test-Path -LiteralPath $binary)){throw 'Portable runtimes are not provisioned in this checkout. Use the Docker Compose instructions in README.md, or provision the project-local Go, PostgreSQL, and Redis cache first.'}
}
foreach($port in @($WebPort,$ApiPort,$PostgresPort,$RedisPort)){if($port -lt 1024 -or $port -gt 65535){throw 'Use ports between 1024 and 65535.'}}
if(-not(Test-Path -LiteralPath (Join-Path $projectRoot 'apps\web\node_modules\next\dist\bin\next'))){throw 'Run npm.cmd ci from apps\web first.'}
function Test-Listening([int]$Port){return [bool](Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)}
function New-LocalSecret { $bytes=New-Object byte[] 48;$rng=[Security.Cryptography.RandomNumberGenerator]::Create();try{$rng.GetBytes($bytes)}finally{$rng.Dispose()};return [Convert]::ToBase64String($bytes).Replace('+','-').Replace('/','_').TrimEnd('=') }
$settingsFile=Join-Path $cacheRoot 'runtime-config.json'
if(-not(Test-Path -LiteralPath $settingsFile)){
 @{database_password=(New-LocalSecret);jwt_secret=(New-LocalSecret);refresh_pepper=(New-LocalSecret)}|ConvertTo-Json|Set-Content -LiteralPath $settingsFile -Encoding UTF8
}
$settings=Get-Content -LiteralPath $settingsFile -Raw|ConvertFrom-Json
$dataRoot=Join-Path $cacheRoot 'data';New-Item -ItemType Directory -Path $dataRoot -Force|Out-Null
$pgData=Join-Path $dataRoot 'postgres'
if(-not(Test-Path -LiteralPath (Join-Path $pgData 'PG_VERSION'))){
 $passwordFile=Join-Path $cacheRoot 'postgres-password.txt';[IO.File]::WriteAllText($passwordFile,$settings.database_password)
 & (Join-Path $pgRoot 'initdb.exe') -D $pgData -U klipforge "--pwfile=$passwordFile" -A scram-sha-256 --encoding=UTF8 --locale=C
 if($LASTEXITCODE -ne 0){throw 'PostgreSQL initialization failed.'}
}
if(-not(Test-Listening $PostgresPort)){
 & (Join-Path $pgRoot 'pg_ctl.exe') -D $pgData -l (Join-Path $dataRoot 'postgres.log') -o "-p $PostgresPort -h 127.0.0.1 -c timezone=UTC" -w start
 if($LASTEXITCODE -ne 0){throw 'PostgreSQL did not start.'}
}
if(-not(Test-Listening $RedisPort)){
 Start-Process -FilePath (Join-Path $redisRoot 'redis-server.exe') -ArgumentList @('--bind','127.0.0.1','--port',"$RedisPort",'--save','""','--appendonly','no') -WorkingDirectory $dataRoot -WindowStyle Hidden -RedirectStandardOutput (Join-Path $dataRoot 'redis.log') -RedirectStandardError (Join-Path $dataRoot 'redis-error.log')|Out-Null
 Start-Sleep -Milliseconds 500
}
$env:PGPASSWORD=$settings.database_password
$exists=& (Join-Path $pgRoot 'psql.exe') -h 127.0.0.1 -p $PostgresPort -U klipforge -d postgres -t -A -c "SELECT 1 FROM pg_database WHERE datname='klipforge'"
if($LASTEXITCODE -ne 0){throw 'The selected PostgreSQL server could not be authenticated. No data was changed.'}
if($exists -ne '1'){& (Join-Path $pgRoot 'createdb.exe') -h 127.0.0.1 -p $PostgresPort -U klipforge klipforge;if($LASTEXITCODE -ne 0){throw 'Development database creation failed.'}}
$env:DATABASE_URL='postgres://klipforge:'+ [uri]::EscapeDataString($settings.database_password)+"@127.0.0.1:$PostgresPort/klipforge?sslmode=disable"
$env:MIGRATIONS_DIR=Join-Path $projectRoot 'migrations'
& $goBinary -C (Join-Path $projectRoot 'services\api') run ./cmd/migrate up
if($LASTEXITCODE -ne 0){throw 'Development database migrations failed.'}
$users=& (Join-Path $pgRoot 'psql.exe') -h 127.0.0.1 -p $PostgresPort -U klipforge -d klipforge -t -A -c 'SELECT count(*) FROM users'
if($users -eq '0'){& $goBinary -C (Join-Path $projectRoot 'services\api') run ./cmd/seed;if($LASTEXITCODE -ne 0){throw 'Development fixture initialization failed.'}}
Remove-Item Env:PGPASSWORD
$env:REDIS_URL="redis://127.0.0.1:$RedisPort/0"
$env:HTTP_HOST='127.0.0.1';$env:PORT="$ApiPort";$env:APP_ENV='development'
$env:JWT_SECRET=$settings.jwt_secret;$env:REFRESH_TOKEN_PEPPER=$settings.refresh_pepper
# Import only the optional server key from the local dotenv file; never execute its contents.
if(-not $env:YOUTUBE_API_KEY){
 $localEnv=Join-Path $projectRoot '.env'
 if(Test-Path -LiteralPath $localEnv){
  foreach($entry in Get-Content -LiteralPath $localEnv){if($entry -match '^\s*YOUTUBE_API_KEY\s*=\s*(.*?)\s*$'){$env:YOUTUBE_API_KEY=$Matches[1].Trim().Trim('"').Trim("'");break}}
 }
}
$env:METRICS_PROVIDER='mock';$env:CORS_ALLOWED_ORIGINS="http://localhost:$WebPort,http://127.0.0.1:$WebPort"
$env:API_INTERNAL_URL="http://127.0.0.1:$ApiPort";$env:NEXT_TELEMETRY_DISABLED='1'
if(-not(Test-Listening $ApiPort)){
 $binRoot=Join-Path $cacheRoot 'bin';New-Item -ItemType Directory -Path $binRoot -Force|Out-Null
 $apiBinary=Join-Path $binRoot 'api-local.exe'
 & $goBinary -C (Join-Path $projectRoot 'services\api') build -o $apiBinary ./cmd/api
 if($LASTEXITCODE -ne 0){throw 'API build failed.'}
 Start-Process -FilePath $apiBinary -WorkingDirectory (Join-Path $projectRoot 'services\api') -WindowStyle Hidden -RedirectStandardOutput (Join-Path $dataRoot 'api.log') -RedirectStandardError (Join-Path $dataRoot 'api-error.log')|Out-Null
}
# Use Webpack for local development: Turbopack HMR can loop on hash mismatches.
if(-not(Test-Listening $WebPort)){
 $webRoot=Join-Path $projectRoot 'apps\web';$entry=Join-Path $webRoot 'node_modules\next\dist\bin\next'
 Start-Process -FilePath (Get-Command node).Source -ArgumentList @(('"'+$entry+'"'),'dev','--webpack','--hostname','127.0.0.1','--port',"$WebPort") -WorkingDirectory $webRoot -WindowStyle Hidden -RedirectStandardOutput (Join-Path $dataRoot 'web.log') -RedirectStandardError (Join-Path $dataRoot 'web-error.log')|Out-Null
}
$healthy=$false
for($attempt=0;$attempt -lt 30;$attempt++){
 try{$health=Invoke-RestMethod -Uri "http://127.0.0.1:$ApiPort/api/v1/health" -TimeoutSec 2;if($health.status -eq 'ok' -and $health.service -eq 'klipforge-api'){$healthy=$true;break}}catch{}
 Start-Sleep -Milliseconds 500
}
if(-not $healthy){throw 'API health check failed. Check .cache/data/api.log and api-error.log.'}
$webHealthy=$false
for($attempt=0;$attempt -lt 30;$attempt++){
 try{$bridge=Invoke-RestMethod -Uri "http://127.0.0.1:$WebPort/api/health" -TimeoutSec 5;if($bridge.connected -and $bridge.api.service -eq 'klipforge-api'){$webHealthy=$true;break}}catch{}
 Start-Sleep -Milliseconds 500
}
if(-not $webHealthy){throw 'Web/API bridge did not become ready. Check .cache/data/web.log and web-error.log.'}
Write-Output "KlipForge preview: http://localhost:$WebPort"
Write-Output "Sign in: http://localhost:$WebPort/login"
Write-Output 'Local-only fixtures are documented in docs/database-schema.md. No Git operations or deployment were run.'
