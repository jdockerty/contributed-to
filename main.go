package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var (

	// The static GraphQL query which we need to use in order to fetch the relevant
	// data from the GitHub API.
	query struct {
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

	cacheSize = *flag.Int("cache-size", 10000, "number of items available to cache")
	port      = *flag.String("port", "6000", "port to bind")
)

// Info is returned to the user.
type Info struct {
	Owner string
	URL   string
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

	// In-memory cache creation, this will lock multiple running instances to
	// each process or container. If there is a requirement later on, we can
	// likely move to Redis in order to have a shared cache between multiple
	// instances of the application.
	cache, err := lru.New[string, interface{}](cacheSize)
	if err != nil {
		fmt.Printf("unable to create cache: %s\n", err)
		return
	}
	log.Printf("cache created with %d max entries", cacheSize)

	router := gin.Default()

	router.SetTrustedProxies(nil)

	router.GET("/user/:name", func(c *gin.Context) {

		name := c.Param("name")

		// Some requests can take a long time. Using an LRU cache here means
		// that the first time a request comes in, it may take awhile to sift
		// through all of their merged PRs, but subsequent requests are returned
		// multiple magnitudes faster.
		if cache.Contains(name) {
			log.Printf("cache hit for %s\n", name)

			// We can discard the "ok" here, since we have already checked
			// via Contains.
			data, _ := cache.Get(name)
			c.JSON(http.StatusOK, data)
			return
		}

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

		cache.Add(name, pullRequests)
		log.Printf("%s added to cache for future requests", name)

		c.JSON(http.StatusOK, pullRequests)
	})

	router.Run(":" + port)
}
