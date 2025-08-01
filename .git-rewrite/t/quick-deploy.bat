@echo off
echo Building and deploying Nephoran Intent Operator...
echo.

cd /d "C:\Users\thc1006\Desktop\nephoran-intent-operator\nephoran-intent-operator"

echo Step 1: Building Docker images...
make docker-build
if %ERRORLEVEL% neq 0 (
    echo Docker build failed!
    pause
    exit /b 1
)

echo.
echo Step 2: Deploying to local cluster...
powershell -File local-deploy.ps1 -UseLocalRegistry
if %ERRORLEVEL% neq 0 (
    echo Deployment failed!
    pause
    exit /b 1
)

echo.
echo Step 3: Validating deployment...
powershell -File validate-environment.ps1
if %ERRORLEVEL% neq 0 (
    echo Validation failed!
    pause
    exit /b 1
)

echo.
echo ✅ Deployment completed successfully!
echo.
echo Useful commands:
echo   kubectl get pods -n nephoran-system
echo   kubectl logs -f deployment/nephio-bridge -n nephoran-system
echo   kubectl apply -f my-first-intent.yaml
echo.
pause