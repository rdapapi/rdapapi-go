package main

import (
	"context"
	"fmt"
	"log"
	"os"

	rdapapi "github.com/rdapapi/rdapapi-go"
)

func main() {
	client := rdapapi.NewClient(os.Getenv("RDAPAPI_KEY"))

	entity, err := client.Entity(context.Background(), "GOGL")
	if err != nil {
		log.Fatal(err)
	}

	if entity.Organization != nil {
		fmt.Printf("Organization: %s\n", *entity.Organization)
	}
	if entity.Handle != nil {
		fmt.Printf("Handle: %s\n", *entity.Handle)
	}
	if entity.Email != nil {
		fmt.Printf("Email: %s\n", *entity.Email)
	}
	fmt.Printf("Networks: %d\n", len(entity.Networks))
	fmt.Printf("Autnums: %d\n", len(entity.Autnums))
}
