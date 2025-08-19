@echo off
REM Mock porch executable for testing
echo Mock porch processing: %*

REM Extract intent file path from arguments
set INTENT_FILE=%2
set OUT_DIR=%4

REM Generate a simple YAML output
set FILENAME=%~n2
echo apiVersion: v1 > "%OUT_DIR%\%FILENAME%.yaml"
echo kind: ConfigMap >> "%OUT_DIR%\%FILENAME%.yaml"
echo metadata: >> "%OUT_DIR%\%FILENAME%.yaml"
echo   name: %FILENAME%-output >> "%OUT_DIR%\%FILENAME%.yaml"
echo data: >> "%OUT_DIR%\%FILENAME%.yaml"
echo   processed: "true" >> "%OUT_DIR%\%FILENAME%.yaml"
echo   timestamp: "%DATE% %TIME%" >> "%OUT_DIR%\%FILENAME%.yaml"

exit /b 0