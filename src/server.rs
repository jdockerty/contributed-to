use axum::extract;
use axum::{routing::get, Router};
use clap_verbosity_flag::LevelFilter;
use tracing_log::AsTrace;

pub async fn serve(address: String, port: u16, verbosity: LevelFilter) -> anyhow::Result<()> {
    tracing_subscriber::fmt::fmt()
        .with_max_level(verbosity.as_trace())
        .init();

    let app = Router::new().route("/user/:name", get(contributions));

    let bind_addr = format!("{}:{}", address, port);
    let listener = tokio::net::TcpListener::bind(bind_addr).await?;
    tracing::info!("Listening on {}", listener.local_addr()?);
    axum::serve(listener, app).await?;
    Ok(())
}

async fn contributions(params: extract::Path<String>) -> String {
    let name = params.0;
    format!("Hello, {}!", name)
}
