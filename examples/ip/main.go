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

	ip, err := client.IP(context.Background(), "8.8.8.8")
	if err != nil {
		log.Fatal(err)
	}

	if ip.Name != nil {
		fmt.Printf("Name: %s\n", *ip.Name)
	}
	if ip.StartAddress != nil && ip.EndAddress != nil {
		fmt.Printf("Range: %s - %s\n", *ip.StartAddress, *ip.EndAddress)
	}
	if ip.Country != nil {
		fmt.Printf("Country: %s\n", *ip.Country)
	}
	fmt.Printf("CIDR: %v\n", ip.CIDR)
	fmt.Printf("Status: %v\n", ip.Status)
}
