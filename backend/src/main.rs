use std::sync::Arc;

use dotenvy::dotenv;
use salvo::{
    http::{StatusCode, header},
    logging::Logger,
    prelude::*,
    serve_static::static_embed,
};
use serde::{Deserialize, Serialize};
use tracing::info;

use zzz_backend::{
    config::AppConfig,
    error::{AppError, ErrorResponse},
    export::{ExportFormat, export_to_host},
    filter::Filter,
    github::{client::GitHubClient, parser::parse_github_url},
    static_files::Assets,
    zip::create_zip,
};

#[derive(Debug, Deserialize)]
struct DownloadQuery {
    url: Option<String>,
    ignore: Option<String>,
    include: Option<String>,
    select: Option<String>,
}

#[derive(Debug, Deserialize)]
struct ExportRequest {
    url: Option<String>,
    ignore: Option<Vec<String>>,
    include: Option<Vec<String>>,
    select: Option<Vec<String>>,
    destination: Option<String>,
    format: Option<String>,
}

#[derive(Debug, Serialize)]
struct Capabilities {
    host_export: bool,
}

#[handler]
async fn health(res: &mut Response) {
    res.status_code(StatusCode::OK);
    res.render(Text::Plain("OK"));
}

#[handler]
async fn capabilities(res: &mut Response, depot: &mut Depot) {
    let config = depot
        .get_typed::<Arc<AppConfig>>()
        .expect("app config is installed");
    res.render(Json(Capabilities {
        host_export: config.download_root.is_some(),
    }));
}

#[handler]
async fn tree(req: &mut Request, res: &mut Response, depot: &mut Depot) {
    let query = match req.parse_queries::<DownloadQuery>() {
        Ok(query) => query,
        Err(error) => return respond_error(res, AppError::InvalidUrl(error.to_string())),
    };
    let url = match query.url.filter(|url| !url.trim().is_empty()) {
        Some(url) => url,
        None => return respond_error(res, AppError::InvalidUrl("missing 'url' parameter".into())),
    };
    let target = match parse_github_url(&url) {
        Ok(target) => target,
        Err(error) => return respond_error(res, error),
    };
    let config = depot
        .get_typed::<Arc<AppConfig>>()
        .expect("app config is installed")
        .clone();
    let filter = match build_filter(query.ignore.as_deref(), query.include.as_deref()) {
        Ok(filter) => filter,
        Err(error) => return respond_error(res, error),
    };
    let client = match GitHubClient::new(config)
        .map(|client| client.with_request_token(request_github_token(req)))
    {
        Ok(client) => client,
        Err(error) => return respond_error(res, error),
    };
    match client.list_tree(&target, &filter).await {
        Ok(entries) => res.render(Json(entries)),
        Err(error) => respond_error(res, error),
    }
}

#[handler]
async fn download(req: &mut Request, res: &mut Response, depot: &mut Depot) {
    let query = match req.parse_queries::<DownloadQuery>() {
        Ok(query) => query,
        Err(error) => return respond_error(res, AppError::InvalidUrl(error.to_string())),
    };
    let url = match query.url.filter(|url| !url.trim().is_empty()) {
        Some(url) => url,
        None => return respond_error(res, AppError::InvalidUrl("missing 'url' parameter".into())),
    };
    let target = match parse_github_url(&url) {
        Ok(target) => target,
        Err(error) => return respond_error(res, error),
    };

    let config = depot
        .get_typed::<Arc<AppConfig>>()
        .expect("app config is installed")
        .clone();
    let filter = match build_filter(query.ignore.as_deref(), query.include.as_deref()) {
        Ok(filter) => filter,
        Err(error) => return respond_error(res, error),
    };
    let selection = parse_selection(query.select.as_deref());
    let client = match GitHubClient::new(config.clone())
        .map(|client| client.with_request_token(request_github_token(req)))
    {
        Ok(client) => client,
        Err(error) => return respond_error(res, error),
    };
    let files = match client.collect_files(&target, &filter, &selection).await {
        Ok(files) if !files.is_empty() => files,
        Ok(_) => return respond_error(res, AppError::NoFiles),
        Err(error) => return respond_error(res, error),
    };
    let zip = match create_zip(&files, &client, &config).await {
        Ok(zip) => zip,
        Err(error) => return respond_error(res, error),
    };

    let name = format!(
        "{}-{}.zip",
        target.repo,
        target.reference.as_deref().unwrap_or("default")
    );
    res.status_code(StatusCode::OK);
    res.headers_mut()
        .insert(header::CONTENT_TYPE, "application/zip".parse().unwrap());
    res.headers_mut().insert(
        header::CONTENT_DISPOSITION,
        format!("attachment; filename=\"{}\"", safe_filename(&name))
            .parse()
            .unwrap(),
    );
    let _ = res.write_body(zip);
}

#[handler]
async fn export(req: &mut Request, res: &mut Response, depot: &mut Depot) {
    let request = match req.parse_json::<ExportRequest>().await {
        Ok(request) => request,
        Err(error) => return respond_error(res, AppError::InvalidUrl(error.to_string())),
    };
    let url = match request.url.filter(|url| !url.trim().is_empty()) {
        Some(url) => url,
        None => return respond_error(res, AppError::InvalidUrl("missing 'url' parameter".into())),
    };
    let target = match parse_github_url(&url) {
        Ok(target) => target,
        Err(error) => return respond_error(res, error),
    };
    let config = depot
        .get_typed::<Arc<AppConfig>>()
        .expect("app config is installed")
        .clone();
    let ignore = request.ignore.unwrap_or_default();
    let include = request.include.unwrap_or_default();
    let filter = if ignore.is_empty() && include.is_empty() {
        match Filter::defaults() {
            Ok(filter) => filter,
            Err(error) => return respond_error(res, error),
        }
    } else {
        match Filter::new(&ignore, &include) {
            Ok(filter) => filter,
            Err(error) => return respond_error(res, error),
        }
    };
    let format = match request.format.as_deref() {
        Some("folder") => ExportFormat::Folder,
        Some("zip") | None => ExportFormat::Zip,
        Some(other) => {
            return respond_error(
                res,
                AppError::InvalidUrl(format!("unsupported export format: {other}")),
            );
        }
    };
    let selection = request.select.unwrap_or_default();
    let client = match GitHubClient::new(config.clone())
        .map(|client| client.with_request_token(request_github_token(req)))
    {
        Ok(client) => client,
        Err(error) => return respond_error(res, error),
    };
    let files = match client.collect_files(&target, &filter, &selection).await {
        Ok(files) if !files.is_empty() => files,
        Ok(_) => return respond_error(res, AppError::NoFiles),
        Err(error) => return respond_error(res, error),
    };
    match export_to_host(
        &files,
        &client,
        &config,
        request.destination.as_deref(),
        format,
        &target.repo,
    )
    .await
    {
        Ok(result) => res.render(Json(result)),
        Err(error) => respond_error(res, error),
    }
}

fn build_filter(ignore: Option<&str>, include: Option<&str>) -> Result<Filter, AppError> {
    let parse = |value: Option<&str>| {
        value
            .unwrap_or_default()
            .split(',')
            .map(str::trim)
            .filter(|s| !s.is_empty())
            .map(ToOwned::to_owned)
            .collect::<Vec<_>>()
    };
    let ignore = parse(ignore);
    let include = parse(include);
    if ignore.is_empty() && include.is_empty() {
        Filter::defaults()
    } else {
        Filter::new(&ignore, &include)
    }
}

fn request_github_token(req: &Request) -> Option<String> {
    req.headers()
        .get("x-github-token")
        .and_then(|value| value.to_str().ok())
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}

fn parse_selection(value: Option<&str>) -> Vec<String> {
    value
        .unwrap_or_default()
        .split(',')
        .map(str::trim)
        .filter(|path| !path.is_empty() && !path.contains("..") && !path.starts_with('/'))
        .map(ToOwned::to_owned)
        .collect()
}

fn safe_filename(value: &str) -> String {
    value
        .chars()
        .map(|ch| {
            if ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_' | '.') {
                ch
            } else {
                '-'
            }
        })
        .collect()
}

fn respond_error(res: &mut Response, error: AppError) {
    res.status_code(error.status_code());
    res.render(Json(ErrorResponse {
        error: error.to_string(),
        status: error.status_code().as_u16(),
        hint: error.hint(),
    }));
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();
    let config = Arc::new(AppConfig::from_env());

    let router = Router::new()
        .hoop(Logger::new())
        .hoop(affix_state::inject(config))
        .push(Router::with_path("api/health").get(health))
        .push(Router::with_path("api/capabilities").get(capabilities))
        .push(Router::with_path("api/tree").get(tree))
        .push(Router::with_path("api/download").get(download))
        .push(Router::with_path("api/export").post(export))
        .push(Router::with_path("{**path}").get(static_embed::<Assets>().fallback("index.html")));

    let acceptor = TcpListener::new("0.0.0.0:8080").bind().await;
    info!("zzz listening on http://0.0.0.0:8080");
    Server::new(acceptor).serve(router).await;
    Ok(())
}
