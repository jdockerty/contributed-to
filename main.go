package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Info is returned to the user.
type Info struct {
	Owner string
	URL   string
}

// The static GraphQL query which we need to use in order to fetch the relevant
// data from the GitHub API.
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

// fetchMergedPullRequestsByUser will fetch the merged pull requests for a given
// user from the GitHub API, it initially uses a nil cursor that is then populated
// from recurring calls.
func fetchMergedPullRequestsByUser(ctx context.Context, client *githubv4.Client, name string, variables map[string]interface{}) ([]Info, error) {

	var allRepos []Info
	for {
		if err := client.Query(context.Background(), &query, variables); err != nil {
			fmt.Println(err)
			return nil, err
		}

		for _, repo := range query.User.PullRequests.Nodes {

			if repo.Repository.Owner.Login == name {
				continue
			}

			info := Info{
				Owner: repo.Repository.NameWithOwner,
				URL:   repo.Permalink,
			}

			allRepos = append(allRepos, info)
		}

		if !query.User.PullRequests.PageInfo.HasNextPage {
			break
		}

		variables["mergedPRCursor"] = githubv4.String(query.User.PullRequests.PageInfo.EndCursor)
	}

	return allRepos, nil
}

// getGitHubClient wraps the creation of a GitHub GraphQL client.
func getGitHubClient(ctx context.Context, token string) *githubv4.Client {

	src := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)

	c := oauth2.NewClient(ctx, src)

	return githubv4.NewClient(c)
}

func main() {

	githubToken := os.Getenv("GH_TOKEN_CONTRIBUTED_TO")
	if githubToken == "" {
		fmt.Println("GH_TOKEN_CONTRIBUTED_TO environment variable must be set.")
		return
	}

	client := getGitHubClient(context.Background(), githubToken)

	router := gin.Default()

	router.SetTrustedProxies(nil)

	router.GET("/user/:name", func(c *gin.Context) {

		name := c.Param("name")
		queryVariables := map[string]interface{}{
			"name":           githubv4.String(name),
			"mergedPRCursor": (*githubv4.String)(nil),
		}

		pullRequests, err := fetchMergedPullRequestsByUser(context.Background(), client, name, queryVariables)
		if err != nil {
			msg := fmt.Sprintf("unable to fetch data for %s", name)
			c.JSON(500, gin.H{
				"message": msg,
			})
			return
		}

		c.JSON(http.StatusOK, pullRequests)
	})

	router.Run()
}
