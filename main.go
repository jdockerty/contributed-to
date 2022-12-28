package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/v48/github"
)

func main() {
	client := github.NewClient(nil)

	var PullReqEvents []*github.Event
	opt := &github.ListOptions{}
	for {
		events, res, err := client.Activity.ListEventsPerformedByUser(context.Background(), "jdockerty", true, opt)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, event := range events {
			if event.GetType() != "PullRequestEvent" {
				continue
			}
			PullReqEvents = append(PullReqEvents, event)
		}

		if res.NextPage == 0 {
			break
		}

		opt.Page = res.NextPage

	}

	for _, event := range PullReqEvents {
		fmt.Println(event.Repo.GetName())
	}

}
