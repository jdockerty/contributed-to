use clap::{Parser, Subcommand};
use clap_verbosity_flag::Verbosity;
use contributed::server;

#[derive(Debug, Clone, Parser)]
#[command(author, version, about, long_about = None)]
struct App {
    /// The name of the user(s) to check public contributions for.
    users: Vec<String>,

    #[command(subcommand)]
    command: Option<Command>,

    #[command(flatten)]
    verbose: Verbosity<clap_verbosity_flag::InfoLevel>,
}

#[derive(Debug, Clone, Subcommand)]
enum Command {
    /// Run the contributed server.
    Server {
        /// The address to run the server on.
        #[arg(long, default_value = "localhost")]
        address: String,

        /// The port to run the server on.
        #[arg(long, default_value = "8080")]
        port: u16,
    },
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let app = App::parse();

    match app.command {
        Some(Command::Server { address, port }) => {
            server::serve(address, port, app.verbose.log_level_filter()).await?;
        }
        None => {
            for user in app.users.iter() {
                println!("Checking contributions for {}", user);
            }
        }
    }
    Ok(())
}
