use axum::extract;
use axum::{routing::get, Router};
use clap_verbosity_flag::LevelFilter;
use graphql_client::GraphQLQuery;
use tracing::{debug, info, instrument};

#[allow(clippy::upper_case_acronyms)]
type URI = String;

const GITHUB_API: &str = "https://api.github.com/graphql";
const GITHUB_TOKEN_ENV: &str = "CONTRIBUTED_GITHUB_TOKEN";

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "graphql/github_schema.graphql",
    query_path = "graphql/contributions.graphql",
    variables_derives = "Clone, Debug",
    response_derives = "Clone, Debug"
)]
struct Contributions;

#[instrument]
pub async fn serve(address: String, port: u16) -> anyhow::Result<()> {
    debug!("Reading {} from environment", GITHUB_TOKEN_ENV);
    let github_token = match std::env::var(GITHUB_TOKEN_ENV) {
        Ok(token) => token,
        Err(_) => {
            anyhow::bail!(
                "Please set the {} environment variable to a valid GitHub token",
                GITHUB_TOKEN_ENV
            );
        }
    };

    let app = Router::new().route("/api/user/:name", get(contributions));

    let bind_addr = format!("{}:{}", address, port);
    let listener = tokio::net::TcpListener::bind(bind_addr).await?;
    info!("Listening on {}", listener.local_addr()?);
    axum::serve(listener, app).await?;
    Ok(())
}

#[instrument]
async fn contributions(params: extract::Path<String>) -> String {
    let name = params.0;
    format!("Hello, {}!", name)
}
