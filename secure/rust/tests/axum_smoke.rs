use std::net::SocketAddr;
use std::time::Duration;

#[tokio::test]
async fn health_and_version_endpoints_work() {
    // Build server state
    let cfg = {
        use std::env;
        env::set_var("API_HOST", "127.0.0.1");
        env::set_var("API_PORT", "0"); // ephemeral
        superbuffer::Config::load()
    };

    // Construct server
    // Note: re-export functions/types to make test minimal if needed
    let server = superbuffer::Server::new(cfg).await;

    // Bind ephemeral port
    let app = server.register_routes();
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();

    // Run server in background
    let handle = tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });

    // Probe endpoints
    let client = reqwest::Client::builder().timeout(Duration::from_secs(2)).build().unwrap();
    let base = format!("http://{}", addr);
    let h = client.get(format!("{}/health", base)).send().await.unwrap();
    assert!(h.status().is_success());
    let v = client.get(format!("{}/version", base)).send().await.unwrap();
    assert!(v.status().is_success());

    // Cancel server
    handle.abort();
}
