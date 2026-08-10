use globset::{Glob, GlobSet, GlobSetBuilder};

use crate::error::AppError;

#[derive(Clone, Debug)]
pub struct Filter {
    ignore: GlobSet,
    include: Option<GlobSet>,
}

impl Filter {
    pub fn new(ignore_patterns: &[String], include_patterns: &[String]) -> Result<Self, AppError> {
        let ignore = build_set(ignore_patterns)?;
        let include = if include_patterns.is_empty() {
            None
        } else {
            Some(build_set(include_patterns)?)
        };
        Ok(Self { ignore, include })
    }

    pub fn defaults() -> Result<Self, AppError> {
        Self::new(
            &[
                "node_modules/**".into(),
                ".git/**".into(),
                "dist/**".into(),
                "build/**".into(),
                ".DS_Store".into(),
                ".env".into(),
                ".*ignore".into(),
            ],
            &[],
        )
    }

    pub fn should_ignore(&self, path: &str) -> bool {
        self.include.as_ref().is_some_and(|set| !set.is_match(path)) || self.ignore.is_match(path)
    }

    pub fn should_prune_directory(&self, path: &str) -> bool {
        self.ignore.is_match(path) || self.ignore.is_match(format!("{path}/placeholder"))
    }
}

fn build_set(patterns: &[String]) -> Result<GlobSet, AppError> {
    let mut builder = GlobSetBuilder::new();
    for pattern in patterns {
        let normalized = pattern.trim().trim_end_matches('/');
        if normalized.is_empty() {
            continue;
        }
        let glob =
            if pattern.ends_with('/') || !normalized.contains('/') && !normalized.contains('.') {
                format!("{normalized}/**")
            } else {
                normalized.to_string()
            };
        builder
            .add(Glob::new(&glob).map_err(|e| AppError::InvalidFilter(format!("{pattern}: {e}")))?);
    }
    builder
        .build()
        .map_err(|e| AppError::InvalidFilter(e.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_globs_and_keeps_invalid_rules_visible() {
        let filter = Filter::new(&["node_modules/".into(), "*.log".into()], &[]).unwrap();
        assert!(filter.should_ignore("node_modules/package.json"));
        assert!(filter.should_ignore("logs/app.log"));
        assert!(!filter.should_ignore("src/main.rs"));
        assert!(Filter::new(&["[".into()], &[]).is_err());
    }
}
