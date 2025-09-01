package main

import (
	"fmt"
	"helm.sh/helm/v3/pkg/cli"
)

func main() {
	settings := cli.New()
	fmt.Println("Helm CLI import working:", settings != nil)
}
