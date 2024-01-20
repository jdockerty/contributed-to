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

    loop {
        let response = Contributions::build_query(variables.clone());
        let response: octocrab::Result<graphql_client::Response<contributions::ResponseData>> =
            client.graphql(&response).await;
        match response {
            Ok(response) => {
                //let response = response.data.expect("missing response data");
                //let contributions = response.user.contributions_collection.contribution_calendar
                //    .contributions;
                //if contributions.is_empty() {
                //    break contributions;
                //}
                //variables.cursor = Some(contributions.last().unwrap().as_ref().unwrap().cursor.clone());
                //debug!("Fetched {} contributions", contributions.len());
            }
            Err(err) => {
                format!("Failed to fetch contributions for {}: {}", user, err);
            }
        }
    }
    format!("Hello, {}!", user)
}
