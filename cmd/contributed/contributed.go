package main

import (
	"flag"
	"fmt"
)

var (
	url string
)

const (
    // Public endpoint is the application which is ran by me, although
    // if anyone wants to run their own then they can alter the destination
    // url here.
	publicEndpoint = "https://api.contributed.jdocklabs.co.uk/user/"
)

func main() {

	flag.StringVar(&url, "url", publicEndpoint, "url that the service is running on, defaults to the public endpoint.")

	flag.Parse()

	fmt.Println(url)
}
