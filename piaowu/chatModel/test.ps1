$urlBase = "http://localhost:8082/api/ai"

function Test-AIAsync($id, $txt) {
    # 1. Start Async
    $data = @{ conversation_id = $id; content = $txt; actor_id = 101 }
    $json = $data | ConvertTo-Json -Compress
    
    Write-Host "Sending Async Request..."
    $resp = Invoke-RestMethod -Method Post -Uri "$urlBase/process" -Body ([System.Text.Encoding]::UTF8.GetBytes($json)) -ContentType "application/json; charset=utf-8"
    $jobID = $resp.job_id
    Write-Host "Job Created: $jobID"

    # 2. Polling
    Write-Host "Polling status..."
    for ($i = 0; $i -lt 15; $i++) {
        $status = Invoke-RestMethod -Method Get -Uri "$urlBase/job/status?job_id=$jobID"
        Write-Host "Current Status: $($status.status)"
        if ($status.status -eq "completed") {
            Write-Host "Success!"
            $status.result | ConvertFrom-Json | Format-List
            return
        }
        if ($status.status -eq "failed") {
            Write-Host "Failed! Error: $($status.error)"
            return
        }
        Start-Sleep -Seconds 2
    }
    Write-Host "Timeout."
}

Test-AIAsync 5001 "I want to complain about the slow refund process!"
