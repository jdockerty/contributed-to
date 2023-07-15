package contributed

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	// HTTP header to force cache invalidation
	CacheRefreshHeader = "X-Contributed-Cache-Refresh"
)

var (

	// The static GraphQL query which we need to use in order to fetch the relevant
	// data from the GitHub API.
	query struct {
		User struct {
			PullRequests struct {
				Nodes []struct {
					Title      string
					Permalink  string
					Repository struct {
						Name  string
						Owner struct {
							Login     string
							AvatarURL string
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

	cacheSize int
	port      string
)

// PullRequest defines a GitHub pull request.
type PullRequest struct {
	Title string
	URL   string
}

// Repository is a GitHub repository. As an example for this project,
// contributed-to is the repository.
type Repository struct {
	Name         string
	PullRequests []PullRequest
}

// Contribution is a merged pull request to a specific repository, or repositories, owned
// by a GitHub user, that the desired author has contributed to.
//
// When utilised with the cache of GitHub user name it is a mapping of the specified user to their
// successful contributions to other organisation's or repository owners project's.
//
// For example:
//
//	{
//	  "google": {
//	    "avatarURL": "...",
//	    "pullRequests": {
//	      "go-jsonnet": {
//	        "PR Title 1": "URL 1",
//	        "PR Title 2": "URL 2"
//	      }
//	    }
//	  },
//	  "hashicorp": {
//	    "avatarURL": "...",
//	    "pullRequests": {
//	      "nomad": {
//	        "PR Title 1": " URL 1"
//	      }
//	    }
//	  }
//	}
type Contribution struct {
	Owner     string
	AvatarURL string
	Repos     []Repository
}

// FetchMergedPullRequestsByUser will fetch the merged pull requests for a given
// user from the GitHub API, it initially uses a nil cursor that is then populated
// from recurring calls.
func FetchMergedPullRequestsByUser(ctx context.Context, client *githubv4.Client, name string, variables map[string]interface{}) ([]Contribution, error) {

	var contributions []Contribution

	indexMapping := make(map[string]int)
	contributionIndex := 0

	for {

		if err := client.Query(context.Background(), &query, variables); err != nil {
			fmt.Println(err)
			return nil, err
		}

		for _, node := range query.User.PullRequests.Nodes {

			if node.Repository.Owner.Login == name {
				continue
			}

			// This contribution does not exist yet, so we can
			// initialise it.
			if index, ok := indexMapping[node.Repository.Owner.Login]; !ok {
				contribution := Contribution{
					Owner:     node.Repository.Owner.Login,
					AvatarURL: node.Repository.Owner.AvatarURL,
				}

				repo := Repository{
					Name: node.Repository.Name,
				}

				pr := PullRequest{
					Title: node.Title,
					URL:   node.Permalink,
				}

				repo.PullRequests = append(repo.PullRequests, pr)

				contribution.Repos = append(contribution.Repos, repo)
				contributions = append(contributions, contribution)

				indexMapping[contribution.Owner] = contributionIndex
				contributionIndex += 1

			} else {
				con := contributions[index]

				// We need to find whether the current repository exists.
				// If it does, we do not want to duplicates its record,
				// and must instead append a pull request to it.
				repoIndex := func(repos []Repository, target string) int {
					for i, r := range repos {
						if r.Name == target {
							return i
						}
					}

					return -1
				}(con.Repos, node.Repository.Name)

				// If the current repo does not exist
				if repoIndex == -1 {
					repo := Repository{
						Name: node.Repository.Name,
					}

					pr := PullRequest{
						Title: node.Title,
						URL:   node.Permalink,
					}

					repo.PullRequests = append(repo.PullRequests, pr)

					con.Repos = append(con.Repos, repo)
					contributions[index] = con
				} else {
					repo := con.Repos[repoIndex]

					pr := PullRequest{
						Title: node.Title,
						URL:   node.Permalink,
					}

					repo.PullRequests = append(repo.PullRequests, pr)
					con.Repos[repoIndex] = repo
					contributions[index] = con
				}
			}
		}

		// No more pull requests available
		if !query.User.PullRequests.PageInfo.HasNextPage {
			break
		}

		// Update cursor for next page
		variables["mergedPRCursor"] = githubv4.String(query.User.PullRequests.PageInfo.EndCursor)
	}

	return contributions, nil
}

// GetGitHubClient wraps the creation of a GitHub GraphQL client.
func GetGitHubClient(ctx context.Context, token string) *githubv4.Client {

	src := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)

	c := oauth2.NewClient(ctx, src)

	return githubv4.NewClient(c)
}
