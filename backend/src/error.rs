use salvo::http::StatusCode;
use serde::Serialize;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    #[error("invalid GitHub URL: {0}")]
    InvalidUrl(String),
    #[error("invalid filter rule: {0}")]
    InvalidFilter(String),
    #[error("GitHub returned {status}: {message}")]
    GitHub { status: u16, message: String },
    #[error("download limit exceeded: {0}")]
    Limit(String),
    #[error("no files matched the filter")]
    NoFiles,
    #[error("server configuration error: {0}")]
    Configuration(String),
    #[error(transparent)]
    Other(#[from] anyhow::Error),
}

impl AppError {
    pub fn status_code(&self) -> StatusCode {
        match self {
            Self::InvalidUrl(_) | Self::InvalidFilter(_) => StatusCode::BAD_REQUEST,
            Self::GitHub { status, .. } => match *status {
                401 => StatusCode::UNAUTHORIZED,
                403 => StatusCode::FORBIDDEN,
                404 => StatusCode::NOT_FOUND,
                429 => StatusCode::TOO_MANY_REQUESTS,
                _ => StatusCode::BAD_GATEWAY,
            },
            Self::Limit(_) => StatusCode::PAYLOAD_TOO_LARGE,
            Self::NoFiles => StatusCode::NOT_FOUND,
            Self::Configuration(_) => StatusCode::SERVICE_UNAVAILABLE,
            Self::Other(_) => StatusCode::INTERNAL_SERVER_ERROR,
        }
    }
}

#[derive(Debug, Serialize)]
pub struct ErrorResponse {
    pub error: String,
    pub status: u16,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hint: Option<&'static str>,
}

impl AppError {
    pub fn hint(&self) -> Option<&'static str> {
        match self {
            Self::GitHub { status: 401, .. } => {
                Some("GitHub Token 无效或已过期，请检查服务端的 GITHUB_TOKEN 配置。")
            }
            Self::GitHub {
                status: 403,
                message,
            } if message.to_ascii_lowercase().contains("rate limit") => {
                Some("GitHub API 已达到速率限制，请在服务端配置 GITHUB_TOKEN 后重试。")
            }
            Self::GitHub { status: 403, .. } => Some(
                "GitHub 拒绝了请求，常见原因是 API 限流或仓库权限不足；请检查服务端的 GITHUB_TOKEN。",
            ),
            Self::GitHub { status: 404, .. } => {
                Some("仓库、分支或路径不存在，私有仓库还可能是当前 GITHUB_TOKEN 无权访问。")
            }
            Self::GitHub { status: 429, .. } => {
                Some("GitHub API 已达到速率限制，请稍后重试或配置服务端 GITHUB_TOKEN。")
            }
            _ => None,
        }
    }
}
