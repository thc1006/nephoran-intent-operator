Write-Host "Testing NL to Intent Endpoint" -ForegroundColor Green

Write-Host "`nTest 1: Valid scaling intent"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "scale nf-sim to 4 in ns ran-a" 2>$null

Write-Host "`nTest 2: Default namespace"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "scale my-app to 3" 2>$null

Write-Host "`nTest 3: Exceeds max replicas (should fail)"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "scale app to 150 in ns test" 2>$null

Write-Host "`nTest 4: Zero replicas (should fail)"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "scale app to 0 in ns test" 2>$null

Write-Host "`nTest 5: Invalid command (should fail)"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "invalid command text" 2>$null

Write-Host "`nTest 6: Deploy intent"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "deploy nginx in ns production" 2>$null

Write-Host "`nTest 7: Delete intent"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "delete old-app from ns staging" 2>$null

Write-Host "`nTest 8: Update config intent"
curl.exe -X POST "http://localhost:8090/nl/intent" -H "Content-Type: text/plain" -d "update myapp set replicas=5 in ns prod" 2>$null

Write-Host "`nAll tests completed!" -ForegroundColor Green