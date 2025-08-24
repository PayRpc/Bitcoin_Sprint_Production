Write-Host "Starting Bitcoin Core RPC Simulator..." -ForegroundColor Yellow

# Fake blockchain state
$height = 850000    # realistic block height
$chain = "main"
$lastBlockTime = Get-Date
$bestHash = [System.Guid]::NewGuid().ToString("N")

# Start simple HTTP listener on 8332
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://127.0.0.1:8332/")
$listener.Start()
Write-Host "RPC Simulator running at http://127.0.0.1:8332" -ForegroundColor Green
Write-Host "CTRL+C to exit" -ForegroundColor DarkGray

try {
    while ($true) {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response

        # Parse incoming request body
        $body = New-Object IO.StreamReader($request.InputStream)
        $json = $body.ReadToEnd() | ConvertFrom-Json
        $method = $json.method

        # Simulate block every 30 seconds for demo
        if ((Get-Date) - $lastBlockTime -gt (New-TimeSpan -Seconds 30)) {
            $height++
            $bestHash = [System.Guid]::NewGuid().ToString("N")
            $lastBlockTime = Get-Date
            Write-Host "New block $height`: $($bestHash.Substring(0,8))..." -ForegroundColor Cyan
        }

        # Respond only to getblockchaininfo
        if ($method -eq "getblockchaininfo") {
            $result = @{
                chain = $chain
                blocks = $height
                bestblockhash = $bestHash
                difficulty = 5000000000000
                mediantime = [int][double]::Parse((Get-Date -UFormat %s))
            }
            $reply = @{
                result = $result
                error  = $null
                id     = $json.id
            } | ConvertTo-Json -Compress
        }
        elseif ($method -eq "getmempoolinfo") {
            $result = @{ size = Get-Random -Minimum 1000 -Maximum 5000 }
            $reply = @{ result = $result; error = $null; id = $json.id } | ConvertTo-Json -Compress
        }
        else {
            $reply = @{ result = $null; error = "Method not implemented"; id = $json.id } | ConvertTo-Json -Compress
        }

        # Set response headers
        $response.ContentType = "application/json"
        $response.StatusCode = 200
        
        $buffer = [System.Text.Encoding]::UTF8.GetBytes($reply)
        $response.ContentLength64 = $buffer.Length
        $response.OutputStream.Write($buffer, 0, $buffer.Length)
        $response.OutputStream.Close()
    }
}
finally {
    $listener.Stop()
    Write-Host "RPC Simulator stopped" -ForegroundColor Red
}
