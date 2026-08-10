use std::path::{Component, Path, PathBuf};

use serde::Serialize;
use tokio::fs;

use crate::{
    config::AppConfig,
    error::AppError,
    github::client::{FileEntry, GitHubClient},
    zip::create_zip,
};

#[derive(Debug, Clone, Copy)]
pub enum ExportFormat {
    Folder,
    Zip,
}

#[derive(Debug, Serialize)]
pub struct ExportResult {
    pub format: &'static str,
    pub path: String,
    pub file_count: usize,
    pub total_size: u64,
}

pub async fn export_to_host(
    files: &[FileEntry],
    client: &GitHubClient,
    config: &AppConfig,
    destination: Option<&str>,
    format: ExportFormat,
    repo_name: &str,
) -> Result<ExportResult, AppError> {
    let root = config
        .download_root
        .as_ref()
        .ok_or_else(|| AppError::Configuration("DOWNLOAD_ROOT is not configured".into()))?;
    fs::create_dir_all(root)
        .await
        .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
    let root = fs::canonicalize(root)
        .await
        .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
    let destination = destination
        .filter(|value| !value.trim().is_empty())
        .unwrap_or(repo_name);
    let relative_destination = safe_relative_path(destination)?;

    match format {
        ExportFormat::Folder => {
            let output = root.join(&relative_destination);
            fs::create_dir_all(&output)
                .await
                .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
            let output = ensure_inside_root(&root, &output).await?;
            let mut total_size = 0_u64;
            for file in files {
                let content = client.download_file(&file.content_url, file.size).await?;
                total_size = total_size.saturating_add(content.len() as u64);
                let path = safe_join(&output, &file.path)?;
                if let Some(parent) = path.parent() {
                    fs::create_dir_all(parent)
                        .await
                        .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
                    ensure_inside_root(&output, parent).await?;
                }
                reject_symlink(&path).await?;
                fs::write(&path, content)
                    .await
                    .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
            }
            Ok(ExportResult {
                format: "folder",
                path: relative_destination.to_string_lossy().into_owned(),
                file_count: files.len(),
                total_size,
            })
        }
        ExportFormat::Zip => {
            let mut relative_file = relative_destination;
            if relative_file.extension().is_none() {
                relative_file.set_extension("zip");
            }
            let output = root.join(&relative_file);
            if let Some(parent) = output.parent() {
                fs::create_dir_all(parent)
                    .await
                    .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
                ensure_inside_root(&root, parent).await?;
            }
            let zip = create_zip(files, client, config).await?;
            reject_symlink(&output).await?;
            fs::write(&output, &zip)
                .await
                .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
            Ok(ExportResult {
                format: "zip",
                path: relative_file.to_string_lossy().into_owned(),
                file_count: files.len(),
                total_size: zip.len() as u64,
            })
        }
    }
}

fn safe_relative_path(value: &str) -> Result<PathBuf, AppError> {
    let path = Path::new(value.trim());
    if path.as_os_str().is_empty()
        || path.components().any(|component| {
            matches!(
                component,
                Component::RootDir | Component::Prefix(_) | Component::ParentDir
            )
        })
    {
        return Err(AppError::InvalidUrl(
            "output path must be relative and cannot contain '..'".into(),
        ));
    }
    Ok(path.to_path_buf())
}

fn safe_join(root: &Path, relative: &str) -> Result<PathBuf, AppError> {
    let relative = safe_relative_path(relative)?;
    Ok(root.join(relative))
}

async fn ensure_inside_root(root: &Path, path: &Path) -> Result<PathBuf, AppError> {
    let canonical = fs::canonicalize(path)
        .await
        .map_err(|error| AppError::Other(anyhow::Error::new(error)))?;
    if !canonical.starts_with(root) {
        return Err(AppError::InvalidUrl(
            "output path escapes DOWNLOAD_ROOT".into(),
        ));
    }
    Ok(canonical)
}

async fn reject_symlink(path: &Path) -> Result<(), AppError> {
    if let Ok(metadata) = fs::symlink_metadata(path).await
        && metadata.file_type().is_symlink()
    {
        return Err(AppError::InvalidUrl(
            "output path cannot be a symbolic link".into(),
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::safe_relative_path;

    #[test]
    fn accepts_relative_output_paths() {
        assert_eq!(
            safe_relative_path("ebooks/2025").unwrap().to_string_lossy(),
            "ebooks/2025"
        );
    }

    #[test]
    fn rejects_paths_that_can_escape_the_root() {
        assert!(safe_relative_path("../outside").is_err());
        assert!(safe_relative_path("/outside").is_err());
    }
}
