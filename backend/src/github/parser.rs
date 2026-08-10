use url::Url;

use crate::error::AppError;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GitHubPath {
    pub owner: String,
    pub repo: String,
    pub reference: Option<String>,
    pub path: String,
    pub is_file: bool,
}

pub fn parse_github_url(input: &str) -> Result<GitHubPath, AppError> {
    let url = Url::parse(input).map_err(|error| AppError::InvalidUrl(error.to_string()))?;
    if url.scheme() != "https" || !matches!(url.host_str(), Some("github.com" | "www.github.com")) {
        return Err(AppError::InvalidUrl(
            "only https://github.com URLs are supported".into(),
        ));
    }
    if url.query().is_some() || url.fragment().is_some() {
        return Err(AppError::InvalidUrl(
            "query strings and fragments are not supported".into(),
        ));
    }

    let segments: Vec<_> = url
        .path_segments()
        .ok_or_else(|| AppError::InvalidUrl("missing repository path".into()))?
        .filter(|segment| !segment.is_empty())
        .collect();
    if segments.len() < 2 {
        return Err(AppError::InvalidUrl("expected /owner/repository".into()));
    }

    let owner = validate_segment(segments[0], "owner")?;
    let repo = validate_segment(segments[1].trim_end_matches(".git"), "repository")?;

    if segments.len() == 2 {
        return Ok(GitHubPath {
            owner,
            repo,
            reference: None,
            path: String::new(),
            is_file: false,
        });
    }

    let kind = segments[2];
    if kind != "blob" && kind != "tree" {
        return Err(AppError::InvalidUrl("expected /blob/ or /tree/".into()));
    }
    if segments.len() < 4 {
        return Err(AppError::InvalidUrl(
            "missing branch, tag, or commit".into(),
        ));
    }

    // GitHub's web URL is ambiguous for refs containing '/'. The first segment
    // is intentionally used here; callers can still use tags and ordinary refs.
    let reference = validate_ref(segments[3])?;
    if segments[4..]
        .iter()
        .any(|segment| *segment == "." || *segment == "..")
    {
        return Err(AppError::InvalidUrl(
            "path traversal segments are not supported".into(),
        ));
    }
    let path = segments[4..].join("/");
    Ok(GitHubPath {
        owner,
        repo,
        reference: Some(reference),
        path,
        is_file: kind == "blob",
    })
}

fn validate_segment(value: &str, label: &str) -> Result<String, AppError> {
    if value.is_empty() || value == "." || value == ".." || value.contains(['/', '\\']) {
        return Err(AppError::InvalidUrl(format!("invalid {label}")));
    }
    Ok(value.to_string())
}

fn validate_ref(value: &str) -> Result<String, AppError> {
    if value.is_empty() || value.contains(['?', '#', '\\']) {
        return Err(AppError::InvalidUrl(
            "invalid branch, tag, or commit".into(),
        ));
    }
    Ok(value.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_file_url() {
        assert_eq!(
            parse_github_url("https://github.com/rust-lang/rust/blob/master/src/main.rs").unwrap(),
            GitHubPath {
                owner: "rust-lang".into(),
                repo: "rust".into(),
                reference: Some("master".into()),
                path: "src/main.rs".into(),
                is_file: true,
            }
        );
    }

    #[test]
    fn parses_root_without_assuming_main() {
        let target = parse_github_url("https://github.com/rust-lang/rust").unwrap();
        assert_eq!(target.reference, None);
        assert!(!target.is_file);
    }

    #[test]
    fn rejects_other_hosts() {
        assert!(parse_github_url("https://github.com.evil.test/a/b").is_err());
    }
}
