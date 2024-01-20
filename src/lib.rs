pub mod server;

#[derive(Debug, Clone)]
pub struct ServerConfig {
    pub address: String,
    pub port: u16,
    pub state: AppState,
}

#[derive(Debug, Clone)]
pub struct AppState {
    pub github_client: octocrab::Octocrab,
}
