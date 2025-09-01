// netkit.rs
// Drop-in networking helpers for Bitcoin Sprint
// - Happy-Eyeballs dial (fast IPv4/IPv6)
// - Tuned TCP (nodelay, keepalive, user-timeout*)
// - TLS connector (rustls, TLS1.3-only, ALPN, session cache)
// - Read/Write deadlines (bound I/O)
// SPDX-License-Identifier: MIT

#![allow(clippy::needless_return)]

use anyhow::{anyhow, Context, Result};
use futures::{stream, StreamExt};
use socket2::{Domain, Socket, TcpKeepalive, Type};
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};
use tokio::net::{lookup_host, TcpStream};

// --- TLS (rustls + tokio-rustls) ---
use rustls::{client::ClientSessionMemoryCache, ClientConfig, RootCertStore};
use rustls::pki_types::ServerName;
use rustls_native_certs;
use tokio_rustls::{client::TlsStream as TokioTlsStream, TlsConnector};

pub type TlsStream = TokioTlsStream<TcpStream>;

// ------------------------------------------------------------
// 1) Happy-Eyeballs connect with tuned socket options
// ------------------------------------------------------------
pub async fn connect_happy(addr: &str, timeout: Duration) -> Result<TcpStream> {
    // addr can be "host:port" or an IP:port
    let addrs: Vec<SocketAddr> =
        lookup_host(addr).await.with_context(|| format!("DNS lookup failed for {}", addr))?
            .collect();

    if addrs.is_empty() {
        return Err(anyhow!("DNS returned no records for {}", addr));
    }

    // Staggered parallel attempts (up to 4 in-flight)
    let attempts = addrs.into_iter().map(move |sa| async move {
        connect_tuned(sa, timeout).await
    });

    let mut s = stream::iter(attempts).buffer_unordered(4);
    while let Some(res) = s.next().await {
        if let Ok(tcp) = res {
            return Ok(tcp);
        }
    }
    Err(anyhow!("All connect attempts failed"))
}

/// Connect a single SocketAddr with tuned TCP options and a bounded timeout.
pub async fn connect_tuned(sa: SocketAddr, timeout: Duration) -> Result<TcpStream> {
    let domain = if sa.is_ipv4() { Domain::IPV4 } else { Domain::IPV6 };
    let socket = Socket::new(domain, Type::STREAM, None).context("create socket")?;
    socket.set_nonblocking(true).context("nonblocking")?;
    socket.set_nodelay(true).context("nodelay")?;

    // Keepalive helps detect dead NATs; values are conservative & cross-platform friendly.
    let ka = TcpKeepalive::new()
        .with_time(Duration::from_secs(30))
        .with_interval(Duration::from_secs(10))
        .with_retries(4);
    socket.set_tcp_keepalive(&ka).ok(); // best-effort across OSes

    // Linux/Android: bound tail by timing out stalled ACKs.
    #[cfg(any(target_os = "linux", target_os = "android"))]
    socket.set_tcp_user_timeout(Some(Duration::from_secs(20))).ok();

    socket
        .connect(&sa.into())
        .or_else(|e| if e.kind() == std::io::ErrorKind::WouldBlock { Ok(()) } else { Err(e) })
        .with_context(|| format!("connect initiate {}", sa))?;

    let stream = TcpStream::from_std(socket.into()).context("adopt std socket")?;

    // Wait until writable or timeout (connect completion)
    tokio::time::timeout(timeout, stream.writable())
        .await
        .context("connect timeout")?
        .context("connect not writable")?;

    Ok(stream)
}

// ------------------------------------------------------------
// 2) TLS: native roots, TLS1.3-only, ALPN(h2/http1), session cache
// ------------------------------------------------------------
pub fn tls_connector(custom_roots: Option<RootCertStore>) -> Result<TlsConnector> {
    // Root store (native by default)
    let roots = if let Some(rs) = custom_roots {
        rs
    } else {
        load_native_roots().context("load native roots")?
    };

    // rustls 0.22 config builder
    #[cfg(feature = "rustls-ring")] // optional, but ring is default provider in many setups
    let provider = rustls::crypto::ring::default_provider();

    #[cfg(not(feature = "rustls-ring"))]
    let provider = rustls::crypto::ring::default_provider(); // keep ring as default even w/o feature

    let mut cfg = ClientConfig::builder_with_provider(provider.into())
        .with_safe_default_protocol_versions()
        .expect("safe default protocol versions")
        .with_root_certificates(roots)
        .with_no_client_auth();

    // ALPN hints (many proxies/CDNs prefer h2 for lower latency)
    cfg.alpn_protocols = vec![b"h2".to_vec(), b"http/1.1".to_vec()];

    // Session cache (resume → fewer handshakes)
    cfg.session_storage = Arc::new(ClientSessionMemoryCache::new(256));

    // TLS 1.3 only (comment out if you must allow TLS 1.2)
    cfg.versions = vec![&rustls::version::TLS13];

    Ok(TlsConnector::from(Arc::new(cfg)))
}

pub async fn connect_tls(domain: &str, port: u16, timeout: Duration) -> Result<TlsStream> {
    let addr = format!("{}:{}", domain, port);
    let tcp = connect_happy(&addr, timeout).await?;
    tcp.set_nodelay(true).ok();

    let connector = tls_connector(None)?;
    let server_name = ServerName::try_from(domain.to_string())
        .map_err(|_| anyhow!("invalid DNS name for SNI: {}", domain))?;

    let tls = connector
        .connect(server_name, tcp)
        .await
        .context("TLS handshake failed")?;

    Ok(tls)
}

fn load_native_roots() -> Result<RootCertStore> {
    let mut store = RootCertStore::empty();
    for cert in rustls_native_certs::load_native_certs().context("native certs")? {
        // Some platforms need a conversion step; try_into handles both cases.
        store
            .add(cert.try_into().map_err(|_| anyhow!("bad cert der"))?)
            .ok();
    }
    Ok(store)
}

// ------------------------------------------------------------
// 3) Bounded I/O helpers (prevent slow peers from stalling tasks)
// ------------------------------------------------------------
pub async fn read_exact_deadline<S>(s: &mut S, buf: &mut [u8], t: Duration) -> Result<()>
where
    S: AsyncRead + Unpin,
{
    tokio::time::timeout(t, s.read_exact(buf))
        .await
        .context("read timeout")?
        .context("read failed")?;
    Ok(())
}

pub async fn write_all_deadline<S>(s: &mut S, buf: &[u8], t: Duration) -> Result<()>
where
    S: AsyncWrite + Unpin,
{
    tokio::time::timeout(t, s.write_all(buf))
        .await
        .context("write timeout")?
        .context("write failed")?;
    Ok(())
}

// ------------------------------------------------------------
// 4) (Optional) frame padding helper to smooth traffic bursts
// ------------------------------------------------------------
pub fn pad_frame(mut msg: Vec<u8>, multiple: usize) -> Vec<u8> {
    if multiple == 0 {
        return msg;
    }
    let pad = (multiple - (msg.len() % multiple)) % multiple;
    if pad > 0 {
        msg.extend(std::iter::repeat(0u8).take(pad));
    }
    msg
}

// ------------------------------------------------------------
// Usage notes (keep for your VS Agent / future reader)
// ------------------------------------------------------------
/*
Cargo.toml (add if not present):

[dependencies]
anyhow = "1"
futures = "0.3"
socket2 = "0.5"
tokio = { version = "1", features = ["full"] }
tokio-rustls = "0.25"
rustls = "0.22"
rustls-native-certs = "0.7"

# (Optional) if you want ring explicitly:
# rustls = { version = "0.22", features = ["ring"] }
# tokio-rustls = { version = "0.25", features = ["ring"] }

--------------------------------------------------------------
Wiring examples:

// 1) Replace direct connects in your P2P client:
let tcp = netkit::connect_happy(&addr, self.cfg.connection_timeout).await?;

// 2) For HTTPS/TLS upstreams:
let mut tls = netkit::connect_tls("api.example.com", 443, Duration::from_secs(10)).await?;
netkit::write_all_deadline(&mut tls, request_bytes, Duration::from_secs(3)).await?;
netkit::read_exact_deadline(&mut tls, &mut buf, Duration::from_secs(3)).await?;

// 3) Optional: smooth frame sizes
let framed = netkit::pad_frame(payload, 128);

--------------------------------------------------------------
Why this single file helps immediately:

- Lower p95/p99 connects → Happy-Eyeballs + tuned sockets
- Fewer stalls → bounded read/write deadlines
- Fewer TLS surprises → proper SNI, native roots, TLS1.3, ALPN
- Faster repeat calls → session resumption
*/
