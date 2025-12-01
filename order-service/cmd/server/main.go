package main

import (
	"log"
	"order-service/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
