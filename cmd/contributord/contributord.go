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

		type htmlData struct {
			User string
		}

		router.LoadHTMLFiles(uiServeFile) // Load templated HTML into renderer

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

			data := &htmlData{
				User: githubUser,
			}

			c.HTML(http.StatusOK, filepath.Base(uiServeFile), data)

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
