// FIXME: Adding package comment per revive linter.

// Package main implements the A1 interface simulator for O-RAN network policy management.

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/a1sim"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/a1/policies", a1sim.SavePolicyHandler("policies"))

	addr := ":8081"

	log.Println("A1 policy sim listening on", addr)

	// Use http.Server with timeouts to fix G114 security warning.

	server := &http.Server{

		Addr: addr,

		Handler: mux,

		ReadTimeout: 15 * time.Second,

		WriteTimeout: 15 * time.Second,

		IdleTimeout: 60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())

}
