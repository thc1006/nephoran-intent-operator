# 重要：設定 MCP 配置檔案

此目錄包含 Model Context Protocol (MCP) 設定檔案的範本。為了保護敏感資訊：

1. **使用範本**：複製 `.mcp.json.example` 為 `.mcp.json`
2. **設定權杖**：將 "your_token_here" 替換為您的實際 GitHub Personal Access Token
3. **保持安全**：絕對不要將包含真實權杖的 `.mcp.json` 檔案提交到 Git

## 檔案說明

- `.mcp.json.example` - 安全的範本檔案
- `.mcp.json` - 您的實際設定（已在 .gitignore 中排除）

## 設定步驟

```powershell
# 複製範本
Copy-Item .mcp.json.example .mcp.json

# 編輯設定檔案，替換 your_token_here 為實際的權杖
```

請確保您的 GitHub Personal Access Token 具有適當的權限以存取您的儲存庫。
