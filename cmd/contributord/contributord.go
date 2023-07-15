package main

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

func printContributions(co []contributed.Contribution) {
	for _, c := range co {
		log.Printf("%+v\n", c.Owner)
	}
}


func main() {
	flag.IntVar(&cacheSize, "cache-size", 1000, "number of items available to cache")
	flag.StringVar(&addr, "address", "localhost", "address to bind")
	flag.StringVar(&port, "port", "6000", "port to bind")
	flag.StringVar(&uiServeFile, "serve-file", "templates/index.html", "path to templated HTML to serve as the frontend")
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
	cache, err := lru.New[string, []contributed.Contribution](cacheSize)
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
		log.Printf("Serving templated UI from %s\n", uiServeFile)

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

			contributions, err := getContributions(githubUser, client, cache)
			if err != nil {
				c.HTML(http.StatusOK, filepath.Base(uiServeFile), nil)
			}

			c.HTML(http.StatusOK, filepath.Base(uiServeFile), contributions)

		})

	}

	router.GET("/api/user/:name", func(c *gin.Context) {

		githubUser := c.Param("name")
		_, ok := c.Request.Header[contributed.CacheRefreshHeader]
		if ok {
			cache.Remove(githubUser)
			log.Printf("%s invalidated from cache", githubUser)
		}

		contributions, err := getContributions(githubUser, client, cache)
		if err != nil {
			msg := fmt.Sprintf("unable to fetch data for %s", githubUser)
			c.JSON(500, gin.H{
				"message": msg,
			})
			return
		}

		cache.Add(githubUser, contributions)
		log.Printf("%s added to cache for future requests", githubUser)

		c.JSON(http.StatusOK, contributions)
	})

	router.Run(fmt.Sprintf("%s:%s", addr, port))
}

func getContributions(githubUser string, client *githubv4.Client, cache *lru.Cache[string, []contributed.Contribution]) ([]contributed.Contribution, error) {

	if cache.Contains(githubUser) {
		// We can discard the "ok" here, since we have already checked
		// via Contains.
		contributions, _ := cache.Get(githubUser)

		return contributions, nil
	}

	queryVariables := map[string]interface{}{
		"name":           githubv4.String(githubUser),
		"mergedPRCursor": (*githubv4.String)(nil),
	}

	contributions, err := contributed.FetchMergedPullRequestsByUser(context.Background(), client, githubUser, queryVariables)
	if err != nil {
		return nil, err
	}

	return contributions, nil
}
