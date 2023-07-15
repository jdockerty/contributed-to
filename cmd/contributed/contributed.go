package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jdockerty/contributed-to/pkg/contributed"
)

const (
	// Public endpoint is the application which is ran by me, although
	// if anyone wants to run their own then they can alter the destination
	// url here.
	publicEndpoint = "https://api.contributed.jdocklabs.co.uk/user"
)

var (
	url          string
	fullInfo     bool
	refreshCache bool
)

func getUser(user string, refreshCache bool) ([]contributed.Contribution, error) {

	c := &http.Client{}

	userEndpoint := fmt.Sprintf("%s/%s", url, user)
	req, err := http.NewRequest("GET", userEndpoint, nil)
	if err != nil {
		return nil, err
	}

	if refreshCache {
		req.Header.Add(contributed.CacheRefreshHeader, "true")
	}

	var contributions []contributed.Contribution

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusInternalServerError {
		return nil, fmt.Errorf("unable to load contributions for %s\n", user)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &contributions)
	if err != nil {
		return nil, err
	}

	return contributions, nil
}

func main() {

	flag.StringVar(&url, "url", publicEndpoint, "url that the service is running on.")
	flag.BoolVar(&fullInfo, "full", false, "display full information about pull requests, e.g. PR title and URL")
	flag.BoolVar(&refreshCache, "refresh", false, "invalidate the given names and refresh the cache")

	flag.Parse()

	// We let the API handle user validation and assume all items passed are
	// valid GitHub users at this point.
	users := flag.CommandLine.Args()

	for _, user := range users {
		contributions, err := getUser(user, refreshCache)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(contributions) == 0 {
			fmt.Printf("%s has no contributions.\n", user)
			return
		}

		fmt.Printf("%s:\n\n", user)

		for _, c := range contributions {
			fmt.Printf("\t%+v\n", c.Owner)

			for _, r := range c.Repos {
				fmt.Printf("\t\t%s\n", r.Name)

				if fullInfo {
					for _, pr := range r.PullRequests {
						fmt.Printf("\t\t\t%s %s\n", pr.Title, pr.URL)
					}
				}
			}
			fmt.Println()
		}
	}
}
