use std::sync::Arc;

use reqwest::{Client, StatusCode};
use serde::{Deserialize, Serialize};
use url::Url;

use crate::{config::AppConfig, error::AppError, filter::Filter, github::parser::GitHubPath};

#[derive(Debug, Clone)]
pub struct FileEntry {
    pub path: String,
    pub content_url: String,
    pub size: u64,
}

#[derive(Debug, Clone, Serialize)]
pub struct TreeEntry {
    pub path: String,
    pub name: String,
    pub kind: String,
    pub size: Option<u64>,
}

#[derive(Debug, Deserialize)]
struct GitTreeResponse {
    tree: Vec<GitTreeItem>,
    truncated: bool,
}

#[derive(Debug, Deserialize)]
struct GitTreeItem {
    path: String,
    #[serde(rename = "type")]
    item_type: String,
    size: Option<u64>,
}

#[derive(Clone)]
pub struct GitHubClient {
    http: Client,
    config: Arc<AppConfig>,
    token: Option<String>,
}

impl GitHubClient {
    pub fn new(config: Arc<AppConfig>) -> Result<Self, AppError> {
        let http = Client::builder()
            .user_agent("zzz/0.1")
            .timeout(config.github_timeout)
            .build()
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        let token = config.github_token.clone();
        Ok(Self {
            http,
            config,
            token,
        })
    }

    pub fn with_request_token(mut self, token: Option<String>) -> Self {
        if token.is_some() {
            self.token = token;
        }
        self
    }

    pub async fn collect_files(
        &self,
        target: &GitHubPath,
        filter: &Filter,
        selection: &[String],
    ) -> Result<Vec<FileEntry>, AppError> {
        let reference = match &target.reference {
            Some(reference) => reference.clone(),
            None => self.default_branch(target).await?,
        };
        let tree = self.get_git_tree(target, &reference).await?;
        if tree.truncated {
            return Err(AppError::Limit(
                "repository tree is too large; choose a narrower directory URL".into(),
            ));
        }

        let mut files = Vec::new();
        let mut declared_total = 0_u64;
        for item in tree.tree {
            if item.item_type != "blob" {
                continue;
            }
            let Some(relative) = tree_relative_path(&target.path, &item.path, target.is_file)
            else {
                continue;
            };
            if filter.should_ignore(&relative) || !selection_matches(selection, &relative) {
                continue;
            }
            let size = item.size.unwrap_or(0);
            if size > self.config.max_file_size {
                return Err(AppError::Limit(format!("{} is too large", item.path)));
            }
            if files.len() >= self.config.max_files {
                return Err(AppError::Limit("too many files".into()));
            }
            if declared_total.saturating_add(size) > self.config.max_total_size {
                return Err(AppError::Limit("total input is too large".into()));
            }
            declared_total = declared_total.saturating_add(size);
            files.push(FileEntry {
                path: relative,
                content_url: raw_download_url(target, &reference, &item.path),
                size,
            });
        }

        if target.is_file && files.is_empty() {
            return Err(AppError::NoFiles);
        }
        Ok(files)
    }

    pub async fn list_tree(
        &self,
        target: &GitHubPath,
        filter: &Filter,
    ) -> Result<Vec<TreeEntry>, AppError> {
        let reference = match &target.reference {
            Some(reference) => reference.clone(),
            None => self.default_branch(target).await?,
        };
        let response = self.get_git_tree(target, &reference).await?;
        if response.truncated {
            return Err(AppError::Limit(
                "repository tree is too large; choose a narrower directory URL".into(),
            ));
        }

        let mut entries = Vec::new();
        for item in response.tree {
            let Some(relative) = tree_relative_path(&target.path, &item.path, target.is_file)
            else {
                continue;
            };
            if relative.is_empty() {
                continue;
            }
            if item.item_type == "tree" {
                if filter.should_prune_directory(&relative) {
                    continue;
                }
                entries.push(TreeEntry {
                    name: entry_name(&relative),
                    path: relative,
                    kind: "dir".into(),
                    size: None,
                });
            } else if item.item_type == "blob" && !filter.should_ignore(&relative) {
                entries.push(TreeEntry {
                    name: entry_name(&relative),
                    path: relative,
                    kind: "file".into(),
                    size: item.size,
                });
            }
            if entries.len() > self.config.max_files.saturating_mul(2) {
                return Err(AppError::Limit("directory tree is too large".into()));
            }
        }

        entries.sort_by(|left, right| left.path.cmp(&right.path));
        Ok(entries)
    }

    pub async fn download_file(&self, url: &str, expected_size: u64) -> Result<Vec<u8>, AppError> {
        let mut request = self.http.get(url);
        if let Some(token) = &self.token {
            request = request.bearer_auth(token);
        }
        let response = request
            .send()
            .await
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(github_error(status, &body));
        }
        let bytes = response
            .bytes()
            .await
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        if bytes.len() as u64 > self.config.max_file_size
            || (expected_size > 0 && bytes.len() as u64 > expected_size.saturating_add(1024 * 1024))
        {
            return Err(AppError::Limit("downloaded file is too large".into()));
        }
        Ok(bytes.to_vec())
    }

    async fn get_git_tree(
        &self,
        target: &GitHubPath,
        reference: &str,
    ) -> Result<GitTreeResponse, AppError> {
        let url = format!(
            "https://api.github.com/repos/{}/{}/git/trees/{}",
            target.owner, target.repo, reference
        );
        let mut request = self.http.get(url).query(&[("recursive", "1")]);
        if let Some(token) = &self.token {
            request = request.bearer_auth(token);
        }
        let response = request
            .send()
            .await
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        let status = response.status();
        let body = response
            .text()
            .await
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        if !status.is_success() {
            return Err(github_error(status, &body));
        }
        serde_json::from_str(&body).map_err(|e| AppError::GitHub {
            status: 502,
            message: e.to_string(),
        })
    }

    async fn default_branch(&self, target: &GitHubPath) -> Result<String, AppError> {
        #[derive(Deserialize)]
        struct Repository {
            default_branch: String,
        }
        let url = format!(
            "https://api.github.com/repos/{}/{}",
            target.owner, target.repo
        );
        let mut request = self.http.get(url);
        if let Some(token) = &self.token {
            request = request.bearer_auth(token);
        }
        let response = request
            .send()
            .await
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        let status = response.status();
        let body = response
            .text()
            .await
            .map_err(|e| AppError::Other(anyhow::Error::new(e)))?;
        if !status.is_success() {
            return Err(github_error(status, &body));
        }
        serde_json::from_str::<Repository>(&body)
            .map(|repo| repo.default_branch)
            .map_err(|e| AppError::GitHub {
                status: 502,
                message: e.to_string(),
            })
    }
}

fn github_error(status: StatusCode, body: &str) -> AppError {
    #[derive(Deserialize)]
    struct GitHubErrorBody {
        message: Option<String>,
    }

    let message = serde_json::from_str::<GitHubErrorBody>(body)
        .ok()
        .and_then(|error| error.message)
        .filter(|message| !message.trim().is_empty())
        .unwrap_or_else(|| body.trim().to_string());

    AppError::GitHub {
        status: status.as_u16(),
        message: message.chars().take(500).collect(),
    }
}

fn entry_name(path: &str) -> String {
    path.rsplit('/').next().unwrap_or(path).to_string()
}

fn tree_relative_path(base: &str, full: &str, is_file: bool) -> Option<String> {
    let base = base.trim_matches('/');
    if base.is_empty() {
        return Some(full.to_string());
    }
    if is_file && full == base {
        return Some(entry_name(full));
    }
    full.strip_prefix(&format!("{}/", base))
        .map(ToOwned::to_owned)
}

fn raw_download_url(target: &GitHubPath, reference: &str, path: &str) -> String {
    let mut url = Url::parse("https://raw.githubusercontent.com/").expect("static URL is valid");
    {
        let mut segments = url
            .path_segments_mut()
            .expect("raw.githubusercontent.com URL has path segments");
        segments.push(&target.owner).push(&target.repo);
        for segment in reference.split('/') {
            segments.push(segment);
        }
        for segment in path.split('/') {
            segments.push(segment);
        }
    }
    url.to_string()
}

fn selection_matches(selection: &[String], path: &str) -> bool {
    selection.is_empty()
        || selection
            .iter()
            .any(|selected| selected == path || path.starts_with(&format!("{selected}/")))
}

#[cfg(test)]
mod tests {
    use super::{raw_download_url, tree_relative_path};
    use crate::github::parser::GitHubPath;

    #[test]
    fn keeps_git_tree_entries_inside_the_selected_path() {
        assert_eq!(
            tree_relative_path("2024", "2024/january/book.pdf", false),
            Some("january/book.pdf".into())
        );
        assert_eq!(
            tree_relative_path("2024/book.pdf", "2024/book.pdf", true),
            Some("book.pdf".into())
        );
        assert_eq!(tree_relative_path("2024", "2023/book.pdf", false), None);
    }

    #[test]
    fn builds_encoded_raw_download_urls() {
        let target = GitHubPath {
            owner: "owner".into(),
            repo: "repo".into(),
            reference: Some("main".into()),
            path: String::new(),
            is_file: false,
        };
        assert_eq!(
            raw_download_url(&target, "main", "books/hello world.pdf"),
            "https://raw.githubusercontent.com/owner/repo/main/books/hello%20world.pdf"
        );
    }
}
