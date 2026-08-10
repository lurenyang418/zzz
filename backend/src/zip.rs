use std::io::{Cursor, Write};

use zip::{CompressionMethod, ZipWriter, write::SimpleFileOptions};

use crate::{
    config::AppConfig,
    error::AppError,
    github::client::{FileEntry, GitHubClient},
};

pub async fn create_zip(
    files: &[FileEntry],
    client: &GitHubClient,
    config: &AppConfig,
) -> Result<Vec<u8>, AppError> {
    let mut writer = ZipWriter::new(Cursor::new(Vec::new()));
    let mut downloaded_total = 0_u64;
    let options = SimpleFileOptions::default()
        .compression_method(CompressionMethod::Deflated)
        .unix_permissions(0o644);

    for file in files {
        let content = client.download_file(&file.content_url, file.size).await?;
        downloaded_total = downloaded_total.saturating_add(content.len() as u64);
        if downloaded_total > config.max_total_size {
            return Err(AppError::Limit(
                "total downloaded content is too large".into(),
            ));
        }
        writer
            .start_file(&file.path, options)
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        writer
            .write_all(&content)
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        if writer
            .get_ref()
            .map(|cursor| cursor.get_ref().len())
            .unwrap_or_default() as u64
            > config.max_zip_size
        {
            return Err(AppError::Limit("generated ZIP is too large".into()));
        }
    }

    let cursor = writer
        .finish()
        .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
    Ok(cursor.into_inner())
}
