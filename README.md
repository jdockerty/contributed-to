# contributed-to

An application which displays contributions, i.e. merged pull requests, that a user has made outside of their own projects.

## Usage

You can make a HTTP request yourself to `api.contributed-to.jdocklabs.co.uk/user/<user>` to receive a JSON payload of information about a particular user.


### CLI

A small CLI is also available which can be installed using

```bash
go install github.com/jdockerty/contributed-to/cmd/contributed@latest
```

By default, this points to `api.contributed-to.jdocklabs.co.uk` but can be altered using the `address` flag if you wish to host the application
on your own infrastructure with specific GitHub tokens.

## How it works

The application runs a simple web server which relays the requested GitHub username from the URL as a parameter into the GitHub API.

This utilises the [GraphQL GitHub API](https://docs.github.com/en/graphql) to fetch the relevant data about a user's merged pull requests, filtering by those which are
not their own projects, i.e. a repository under their GitHub username.

The API call is relatively expensive in that it requests the entire history of merged pull requests until
there are no more remaining that GitHub has kept a record of. It also caches responses, meaning that the first request may take some time but subsequent requests are served
from the cache and are *blazingly* fast.
