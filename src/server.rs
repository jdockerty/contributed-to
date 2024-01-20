use crate::{AppState, ServerConfig};
use axum::extract;
use axum::extract::State;
use axum::{routing::get, Router};
use graphql_client::GraphQLQuery;
use tracing::{debug, info, instrument};

#[allow(clippy::upper_case_acronyms)]
type URI = String;

#[derive(GraphQLQuery)]
#[graphql(
    schema_path = "graphql/github_schema.graphql",
    query_path = "graphql/contributions.graphql",
    variables_derives = "Clone, Debug",
    response_derives = "Clone, Debug"
)]
struct Contributions;

#[instrument]
pub async fn serve(config: ServerConfig) -> anyhow::Result<()> {
    let address = config.address;
    let port = config.port;
    let app = Router::new()
        .route("/api/user/:name", get(contributions))
        .with_state(config.state);

    let bind_addr = format!("{}:{}", address, port);
    let listener = tokio::net::TcpListener::bind(bind_addr).await?;
    info!("Listening on {}", listener.local_addr()?);
    axum::serve(listener, app).await?;
    Ok(())
}

#[instrument]
async fn contributions(State(state): State<AppState>, params: extract::Path<String>) -> String {
    let user = params.0;
    info!("Fetching contributions for {}", user);
    let client = state.github_client;
    let mut variables = contributions::Variables {
        login: user.clone(),
        cursor: None,
    };

    format!("Hello, {}!", user)
}
