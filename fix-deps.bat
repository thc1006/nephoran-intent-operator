@echo off
echo Fixing Go module dependencies...

REM Set environment variables for faster downloads
set GOPROXY=direct
set GOSUMDB=off

REM Clean everything
echo Cleaning module cache...
go clean -modcache 2>nul

echo Updating go.mod...
go mod edit -go=1.24

REM Fix kubernetes dependencies to v0.28.0
echo Updating k8s dependencies to v0.28.0...
go get k8s.io/apimachinery@v0.28.0
go get k8s.io/client-go@v0.28.0
go get k8s.io/api@v0.28.0
go get sigs.k8s.io/controller-runtime@v0.16.0

REM Download specific dependencies
echo Downloading controller-runtime dependencies...
go get sigs.k8s.io/controller-runtime/pkg/client@v0.16.0
go get sigs.k8s.io/controller-runtime/pkg/manager@v0.16.0
go get sigs.k8s.io/controller-runtime/pkg/controller@v0.16.0
go get sigs.k8s.io/controller-runtime/pkg/reconcile@v0.16.0

REM Fix module issues
echo Running go mod tidy...
go mod tidy -v

echo Downloading all dependencies...
go mod download

echo Verifying modules...
go mod verify

echo Done!
pause