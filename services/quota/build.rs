fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = std::path::PathBuf::from(std::env::var("OUT_DIR")?);

    tonic_build::configure()
        .file_descriptor_set_path(out_dir.join("quota_descriptor.bin"))
        .compile_protos(
            &["../../shared/api/quota/v1/quota.proto"],
            &["../../shared/api"],
        )?;

    Ok(())
}
