use bincode::Options;
use solana_entry::entry::Entry;

#[allow(dead_code)]
mod proto {
    pub mod shared {
        include!(concat!(env!("OUT_DIR"), "/shared.rs"));
    }

    pub mod shredstream {
        include!(concat!(env!("OUT_DIR"), "/shredstream.rs"));
    }
}

use proto::shredstream::shredstream_proxy_client::ShredstreamProxyClient;
use proto::shredstream::SubscribeEntriesRequest;

fn decode_entries(bytes: &[u8]) -> Result<Vec<Entry>, bincode::Error> {
    bincode::DefaultOptions::new()
        .with_fixint_encoding()
        .allow_trailing_bytes()
        .with_limit(bytes.len() as u64)
        .deserialize(bytes)
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::args()
        .nth(1)
        .unwrap_or_else(|| "http://127.0.0.1:9900".to_string());
    println!("subscribing to {endpoint}");

    let mut client = ShredstreamProxyClient::connect(endpoint).await?;
    let mut stream = client
        .subscribe_entries(SubscribeEntriesRequest {})
        .await?
        .into_inner();

    while let Some(frame) = stream.message().await? {
        let entries = decode_entries(&frame.entries)?;
        let transactions: usize = entries.iter().map(|entry| entry.transactions.len()).sum();
        let first = entries
            .iter()
            .flat_map(|entry| &entry.transactions)
            .find_map(|tx| tx.signatures.first())
            .map(|signature| signature.to_string())
            .unwrap_or_else(|| "-".to_string());
        println!(
            "slot {:>12}  entries {:>4}  txs {:>4}  first {}",
            frame.slot,
            entries.len(),
            transactions,
            first
        );
    }
    Ok(())
}
