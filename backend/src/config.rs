use std::env;
use std::path::PathBuf;
use std::time::Duration;

#[derive(Clone, Debug)]
pub struct AppConfig {
    pub github_token: Option<String>,
    pub github_timeout: Duration,
    pub max_files: usize,
    pub max_file_size: u64,
    pub max_total_size: u64,
    pub max_zip_size: u64,
    pub download_root: Option<PathBuf>,
}

impl AppConfig {
    pub fn from_env() -> Self {
        Self {
            github_token: env::var("GITHUB_TOKEN")
                .ok()
                .filter(|value| !value.trim().is_empty()),
            github_timeout: Duration::from_secs(env_u64("GITHUB_TIMEOUT_SECS", 60)),
            max_files: env_usize("MAX_FILES", 10_000),
            max_file_size: env_u64("MAX_FILE_SIZE_BYTES", 100 * 1024 * 1024),
            max_total_size: env_u64("MAX_TOTAL_SIZE_BYTES", 512 * 1024 * 1024),
            max_zip_size: env_u64("MAX_ZIP_SIZE_BYTES", 512 * 1024 * 1024),
            download_root: env::var("DOWNLOAD_ROOT")
                .ok()
                .map(PathBuf::from)
                .filter(|path| !path.as_os_str().is_empty()),
        }
    }
}

fn env_u64(name: &str, default: u64) -> u64 {
    env::var(name)
        .ok()
        .and_then(|value| value.parse().ok())
        .unwrap_or(default)
}

fn env_usize(name: &str, default: usize) -> usize {
    env::var(name)
        .ok()
        .and_then(|value| value.parse().ok())
        .unwrap_or(default)
}
