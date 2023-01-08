package contributed

import (
	"context"
	"fmt"

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

// MergedPullRequestInfo contains the relevant information which is fetched from
// the GraphQL query, this is returned to the user.
type MergedPullRequestInfo map[string]PullRequestInfo

// PullRequestInfo represents information about a pull request to a repository
// owner mapping. This holds the avatar URL and merged pull requests together.
type PullRequestInfo struct {

	// AvatarURL is the display picture of the repository owner or organisation.
	AvatarURL string `json:"avatarURL"`

	// PullRequests is the internal representation for the map structure of a
	// owner/organisation to multiple merged pull requests.
	PullRequests PullRequests `json:"pullRequests"`
}

// PullRequests is a custom wrapper around the structure of the response, this
// is a mapping of the repository owner/organisation to the repositories that
// they own, with the merged pull requests contained within the mapping.
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
type PullRequests map[string]map[string]string

// FetchMergedPullRequestsByUser will fetch the merged pull requests for a given
// user from the GitHub API, it initially uses a nil cursor that is then populated
// from recurring calls.
func FetchMergedPullRequestsByUser(ctx context.Context, client *githubv4.Client, name string, variables map[string]interface{}) (map[string]PullRequestInfo, error) {

	mergedPRInfo := make(MergedPullRequestInfo)

	for {
		if err := client.Query(context.Background(), &query, variables); err != nil {
			fmt.Println(err)
			return nil, err
		}

		for _, node := range query.User.PullRequests.Nodes {

			if node.Repository.Owner.Login == name {
				continue
			}

			initPRMap := make(map[string]map[string]string)
			initMap := make(map[string]string)
			// Initialise structure for repository owner to merged requests
			// mapping.
			if _, ok := mergedPRInfo[node.Repository.Owner.Login]; !ok {
				mergedPRInfo[node.Repository.Owner.Login] = PullRequestInfo{
					AvatarURL:    node.Repository.Owner.AvatarURL,
					PullRequests: initPRMap,
				}
			}

			pullRequests := mergedPRInfo[node.Repository.Owner.Login].PullRequests
			if _, ok := pullRequests[node.Repository.Name]; !ok {
				pullRequests[node.Repository.Name] = initMap
			}

			pullRequests[node.Repository.Name][node.Title] = node.Permalink

		}

		// No more pull requests available
		if !query.User.PullRequests.PageInfo.HasNextPage {
			break
		}

		// Update cursor for next page
		variables["mergedPRCursor"] = githubv4.String(query.User.PullRequests.PageInfo.EndCursor)
	}

	return mergedPRInfo, nil
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
