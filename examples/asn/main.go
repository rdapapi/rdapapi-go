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

	asn, err := client.ASN(context.Background(), "AS15169")
	if err != nil {
		log.Fatal(err)
	}

	if asn.Name != nil {
		fmt.Printf("Name: %s\n", *asn.Name)
	}
	if asn.Handle != nil {
		fmt.Printf("Handle: %s\n", *asn.Handle)
	}
	fmt.Printf("Status: %v\n", asn.Status)
}
