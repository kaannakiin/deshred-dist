fn main() -> std::io::Result<()> {
    tonic_prost_build::configure()
        .bytes("shredstream.Entry.entries")
        .compile_protos(&["shredstream.proto"], &["../../proto/jito-shredstream"])
}
