package main

// contributord is the API server and does a bulk of the work.

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jdockerty/contributed-to/pkg/contributed"
	"github.com/shurcooL/githubv4"
)

var (
	cacheSize   int
	port        string
	addr        string
	ui          bool
	uiServeFile string
)

// TODO: clean this up and use the structs here for the actual return data so we
// don't have to do anything special for returning the information and manipulating it.
func buildHtmlData(pullRequests contributed.MergedPullRequestInfo) []pageData {

	var pd []pageData

	for owner, prInfo := range pullRequests {
		// log.Printf("Owner: %s, Info: %+v\n", owner, prInfo)

		data := pageData{
			Owner:     owner,
			AvatarURL: prInfo.AvatarURL,
			Repos:     []Repository{},
		}

		for k, v := range prInfo.PullRequests {
			//log.Printf("Repo: %s, Reqs: %+v\n", k, v)

			r := Repository{
				Name:         k,
				PullRequests: []PullReq{},
			}

			for title, url := range v {
				//log.Printf("Title: %s, URL: %s\n", title, url)
				req := PullReq{
					Title: title,
					URL:   url,
				}
				r.PullRequests = append(r.PullRequests, req)
			}

			data.Repos = append(data.Repos, r)
		}

		pd = append(pd, data)

	}

	return pd

}

type PullReq struct {
	Title string
	URL   string
}
type Repository struct {
	Name         string
	PullRequests []PullReq
}
type pageData struct {
	Owner     string
	AvatarURL string
	Repos     []Repository
}

func main() {
	flag.IntVar(&cacheSize, "cache-size", 1000, "number of items available to cache")
	flag.StringVar(&addr, "address", "localhost", "address to bind")
	flag.StringVar(&port, "port", "6000", "port to bind")
	flag.StringVar(&uiServeFile, "serve-file", "", "path to templated HTML to serve as the frontend")
	flag.Parse()

	githubToken := os.Getenv("GH_TOKEN_CONTRIBUTED_TO")
	if githubToken == "" {
		fmt.Println("GH_TOKEN_CONTRIBUTED_TO environment variable must be set.")
		os.Exit(1)
	}

	client := contributed.GetGitHubClient(context.Background(), githubToken)

	// In-memory cache creation, this will lock multiple running instances to
	// each process or container. If there is a requirement later on, we can
	// likely move to Redis in order to have a shared cache between multiple
	// instances of the application.
	cache, err := lru.New[string, contributed.MergedPullRequestInfo](cacheSize)
	if err != nil {
		fmt.Printf("unable to create cache: %s\n", err)
		return
	}
	log.Printf("cache created with %d max entries", cacheSize)

	router := gin.Default()

	router.SetTrustedProxies(nil)

	router.GET("/api/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	if uiServeFile != "" {

		router.LoadHTMLFiles(uiServeFile) // Load templated HTML into renderer
		router.Static("./static", "static")

		// Initial loading of the page, no value is given
		router.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, filepath.Base(uiServeFile), nil)
		})

		// Form data for the GitHub user is sent as a POST request,
		// so it will land here and re-render the template with some special logic.
		router.POST("/", func(c *gin.Context) {

			githubUser, ok := c.GetPostForm("github_user")
			if !ok {
				c.String(http.StatusBadRequest, "invalid github_user provided")
				return
			}

			if cache.Contains(githubUser) {
				log.Printf("[UI] cache hit for %s\n", githubUser)

				// We can discard the "ok" here, since we have already checked
				// via Contains.
				pullRequests, _ := cache.Get(githubUser)

				htmlData := buildHtmlData(pullRequests)

				c.HTML(http.StatusOK, filepath.Base(uiServeFile), htmlData)
				return
			}

			queryVariables := map[string]interface{}{
				"name":           githubv4.String(githubUser),
				"mergedPRCursor": (*githubv4.String)(nil),
			}

			pullRequests, err := contributed.FetchMergedPullRequestsByUser(context.Background(), client, githubUser, queryVariables)
			if err != nil {
				// show error
				c.HTML(http.StatusOK, filepath.Base(uiServeFile), nil)
				return
			}

			htmlData := buildHtmlData(pullRequests)

			c.HTML(http.StatusOK, filepath.Base(uiServeFile), htmlData)

		})

	}

	router.GET("/api/user/:name", func(c *gin.Context) {

		name := c.Param("name")
		_, ok := c.Request.Header[contributed.CacheRefreshHeader]
		if ok {
			cache.Remove(name)
			log.Printf("%s invalidated from cache", name)
		}

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

		pullRequests, err := contributed.FetchMergedPullRequestsByUser(context.Background(), client, name, queryVariables)
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

	router.Run(fmt.Sprintf("%s:%s", addr, port))
}
