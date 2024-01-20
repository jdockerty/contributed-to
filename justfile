# Update the GraphQL schema from the public GitHub API.
update_graphql_schema:
    curl -L https://docs.github.com/public/schema.docs.graphql -o graphql/github_schema.graphql
