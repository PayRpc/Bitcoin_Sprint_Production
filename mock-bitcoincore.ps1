# Mock Bitcoin Core RPC server (simulation only)
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://127.0.0.1:8332/")
$listener.Start()
Write-Host "Mock Bitcoin RPC server started on 127.0.0.1:8332"

while ($true) {
    try {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response

        # Always respond with blockchain info
        $json = @{
            result = @{
                chain = "regtest"
                blocks = 123456
                bestblockhash = "0000000000000000000abcdef1234567890"
            }
            error = $null
            id = "test"
        } | ConvertTo-Json -Compress

        $buffer = [System.Text.Encoding]::UTF8.GetBytes($json)
        $response.ContentLength64 = $buffer.Length
        $response.OutputStream.Write($buffer, 0, $buffer.Length)
        $response.OutputStream.Close()
    }
    catch {
        Write-Host "Error: $($_.Exception.Message)"
        break
    }
}
