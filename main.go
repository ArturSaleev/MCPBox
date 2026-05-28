package main

import (
	"log"

	"github.com/ArturSaleev/MCPBox/app"
)

func main() {
	if err := app.Run(app.Options{
		Edition: app.FreeEdition(),
	}); err != nil {
		log.Fatalf("mcpbox failed: %v", err)
	}
}
