package main

import (
	"context"
	"fmt"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GH_TOKEN_CONTRIBUTED_TO")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	v := map[string]interface{}{
		"name":           githubv4.String("jdockerty"),
		"mergedPRCursor": (*githubv4.String)(nil),
	}

	type Repository struct {
		NameWithOwner string
		Owner         struct {
			Login string
		}
	}
	type Info struct {
		Owner string
		URL   string
	}
	var query struct {
		User struct {
			PullRequests struct {
				Nodes []struct {
					Permalink  string
					Repository struct {
						NameWithOwner string
						Owner         struct {
							Login string
						}
					}
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"pullRequests(first: 100, after: $mergedPRCursor, states: MERGED)"`
		} `graphql:"user(login: $name)"`
	}

	var allRepos []Info
	for {
		if err := client.Query(context.Background(), &query, v); err != nil {
			fmt.Println(err)
			return
		}

		for _, repo := range query.User.PullRequests.Nodes {
			if repo.Repository.Owner.Login == "jdockerty" {
				continue
			}
			info := Info{
				Owner: repo.Repository.Owner.Login,
				URL:   repo.Permalink,
			}
			allRepos = append(allRepos, info)
		}

		if !query.User.PullRequests.PageInfo.HasNextPage {
			break
		}
		v["mergedPRCursor"] = githubv4.String(query.User.PullRequests.PageInfo.EndCursor)
	}

	for _, r := range allRepos {
		fmt.Println(r.Owner, r.URL)
	}

}
