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

	ns, err := client.Nameserver(context.Background(), "ns1.google.com")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\n", ns.LDHName)
	fmt.Printf("IPv4: %v\n", ns.IPAddresses.V4)
	fmt.Printf("IPv6: %v\n", ns.IPAddresses.V6)
	fmt.Printf("Status: %v\n", ns.Status)
}
