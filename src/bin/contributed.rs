use clap::{Parser, Subcommand};

#[derive(Debug, Clone, Parser)]
#[command(author, version, about, long_about = None)]
struct App {
    /// The name of the user(s) to check public contributions for.
    users: Vec<String>,

    #[command(subcommand)]
    command: Option<Command>,
}

#[derive(Debug, Clone, Subcommand)]
enum Command {
    /// Run the contributed server.
    Server {
        /// The port to run the server on.
        #[arg(long, default_value = "8080")]
        port: u16,

        /// The address to run the server on.
        #[arg(long, default_value = "localhost")]
        address: String,
    },
}

fn main() {
    let app = App::parse();

    match app.command {
        Some(Command::Server { port, address }) => {
            println!("Running server on {}:{}", address, port);
        }
        None => {
            for user in app.users.iter() {
                println!("Checking contributions for {}", user);
            }
        }
    }
}
