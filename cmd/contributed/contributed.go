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
	url      string
	fullInfo bool
)

func getUser(user string) (contributed.MergedPullRequestInfo, error) {
	userEndpoint := fmt.Sprintf("%s/%s", url, user)
	resp, err := http.Get(userEndpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	v := make(contributed.MergedPullRequestInfo)
	err = json.Unmarshal(body, &v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func main() {

	flag.StringVar(&url, "url", publicEndpoint, "url that the service is running on.")
	flag.BoolVar(&fullInfo, "full", false, "display full information about pull requests, e.g. PR title and URL")

	flag.Parse()

	// We let the API handle user validation and assume all items passed are
	// valid GitHub users at this point.
	users := flag.CommandLine.Args()

	for _, user := range users {
		info, err := getUser(user)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("%s has contributed to:\n\n", user)
		for repoOwner, pullRequestInfo := range info {
			fmt.Printf("%s\n", repoOwner)

			for repo, prs := range pullRequestInfo.PullRequests {
				fmt.Printf("\t%s\n", repo)

				if fullInfo {
					for title, url := range prs {
						fmt.Printf("\t\t%s %s\n", title, url)
					}
				}

			}

			// Blank line after each new entry
			fmt.Println()
		}
	}
}
