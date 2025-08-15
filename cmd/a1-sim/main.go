package main

import (
"log"
"net/http"

"github.com/thc1006/nephoran-intent-operator/internal/a1sim"
)

func main() {
mux := http.NewServeMux()
mux.HandleFunc("/a1/policies", a1sim.SavePolicyHandler("policies"))

addr := ":8081"
log.Println("A1 policy sim listening on", addr)
log.Fatal(http.ListenAndServe(addr, mux))
}
