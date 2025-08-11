package main

import (
  "log"
  "net/http"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request){ w.Write([]byte("ok")) })
  // TODO: /intent → 解析/驗證 JSON（依 docs/contracts/intent.schema.json）→ 建 CR
  log.Println("intent-ingest listening on :8080")
  log.Fatal(http.ListenAndServe(":8080", mux))
}
