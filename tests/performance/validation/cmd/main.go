
package main



import (

	"log"

	"os"



	validation "github.com/thc1006/nephoran-intent-operator/tests/performance/validation"

)



// main provides a CLI entry point for the validation suite.

func main() {

	// Set up logging.

	log.SetFlags(log.LstdFlags | log.Lshortfile)



	// Create and execute the validation command.

	cmd := validation.NewValidationCommand()

	if err := cmd.Execute(); err != nil {

		log.Printf("Command failed: %v", err)

		log.Fatal(1)

	}

}

