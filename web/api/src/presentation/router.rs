use axum::{body::Body, http::Request, routing::delete, routing::get, Router};
use http::{HeaderValue, Method};
use std::time::Duration;
use tower_http::{
    cors::{AllowOrigin, CorsLayer},
    request_id::{MakeRequestUuid, PropagateRequestIdLayer, SetRequestIdLayer},
    trace::TraceLayer,
};

use std::sync::Arc;

use crate::{
    application::use_cases::{
        mylist_repository::SdzMyListRepository, spot_repository::SdzSpotRepository,
        storage_repository::SdzStorageRepository, user_repository::SdzUserRepository,
    },
    domain::models::SdzUser,
    infrastructure::{
        firestore_mylist_repository::SdzFirestoreMyListRepository,
        firestore_spot_repository::SdzFirestoreSpotRepository,
        firestore_user_repository::SdzFirestoreUserRepository,
        in_memory_mylist_repository::SdzInMemoryMyListRepository,
        in_memory_spot_repository::SdzInMemorySpotRepository,
        in_memory_user_repository::SdzInMemoryUserRepository,
        storage_disabled_repository::SdzDisabledStorageRepository,
        storage_signed_url_repository::SdzStorageSignedUrlRepository,
    },
};

use super::handlers::{health_handler, mylist_handler, spot_handler, user_handler};

pub fn sdz_build_router() -> Router {
    let state = build_state();

    Router::new()
        .route("/sdz/health", get(health_handler::handle_health))
        .route("/sdz/users/me", get(user_handler::handle_get_me))
        .route(
            "/sdz/spots",
            axum::routing::post(spot_handler::handle_create_spot),
        )
        .route(
            "/sdz/spots/upload-url",
            axum::routing::post(spot_handler::handle_create_upload_url),
        )
        .route("/sdz/spots", get(spot_handler::handle_list_spots))
        .route(
            "/sdz/spots/{spot_id}",
            get(spot_handler::handle_get_spot).patch(spot_handler::handle_update_spot),
        )
        .route(
            "/sdz/admin/spots",
            axum::routing::post(spot_handler::handle_admin_create_spot),
        )
        .route(
            "/sdz/admin/spots/{spot_id}",
            axum::routing::patch(spot_handler::handle_admin_update_spot),
        )
        .route(
            "/sdz/admin/spots/upload-url",
            axum::routing::post(spot_handler::handle_admin_create_upload_url),
        )
        .route(
            "/sdz/mylist",
            get(mylist_handler::handle_list_mylist).post(mylist_handler::handle_add_mylist),
        )
        .route(
            "/sdz/mylist/{spot_id}",
            delete(mylist_handler::handle_remove_mylist),
        )
        .with_state(state)
        .layer(PropagateRequestIdLayer::x_request_id())
        .layer(SetRequestIdLayer::x_request_id(MakeRequestUuid))
        .layer(cors_layer())
        .layer(
            TraceLayer::new_for_http()
                .make_span_with(|request: &Request<Body>| {
                    let request_id = request
                        .extensions()
                        .get::<tower_http::request_id::RequestId>()
                        .and_then(|id| id.header_value().to_str().ok())
                        .unwrap_or("-");
                    tracing::info_span!(
                        "http_request",
                        event_code = "SDZ-API-1001",
                        component = "presentation",
                        method = %request.method(),
                        uri = %request.uri(),
                        request_id = %request_id
                    )
                })
                .on_response(
                    |response: &axum::response::Response,
                     latency: Duration,
                     span: &tracing::Span| {
                        tracing::info!(
                            parent: span,
                            event_code = "SDZ-API-1002",
                            component = "presentation",
                            status = %response.status(),
                            latency_ms = latency.as_millis()
                        );
                    },
                )
                .on_failure(
                    |error: tower_http::classify::ServerErrorsFailureClass,
                     latency: Duration,
                     span: &tracing::Span| {
                        tracing::error!(
                            parent: span,
                            event_code = "SDZ-API-1003",
                            component = "presentation",
                            error = %error,
                            latency_ms = latency.as_millis()
                        );
                    },
                ),
        )
}

fn cors_layer() -> CorsLayer {
    CorsLayer::new()
        .allow_origin(allowed_origins())
        .allow_methods([Method::GET, Method::POST, Method::PATCH, Method::DELETE])
        .allow_headers(tower_http::cors::Any)
}

fn allowed_origins() -> AllowOrigin {
    // 環境変数SDZ_CORS_ALLOWED_ORIGINSにカンマ区切りで指定。
    // 未設定時はローカル開発用のlocalhostを許可。
    if let Ok(origins) = std::env::var("SDZ_CORS_ALLOWED_ORIGINS") {
        let list: Vec<HeaderValue> = origins
            .split(',')
            .filter_map(|s| {
                let trimmed = s.trim();
                if trimmed.is_empty() {
                    None
                } else {
                    HeaderValue::from_str(trimmed).ok()
                }
            })
            .collect();
        if !list.is_empty() {
            return AllowOrigin::list(list);
        }
    }

    // デフォルトは開発用
    AllowOrigin::list(
        ["http://localhost:3000", "http://127.0.0.1:3000"]
            .iter()
            .filter_map(|o| HeaderValue::from_str(o).ok()),
    )
}

fn build_state() -> SdzAppState {
    let storage_repo: Arc<dyn SdzStorageRepository> = build_storage_repo();
    // 環境変数が整っていればFirestore実装を採用
    if std::env::var("SDZ_USE_FIRESTORE").ok().as_deref() == Some("1") {
        if let Ok(project_id) = std::env::var("SDZ_FIRESTORE_PROJECT_ID")
            .or_else(|_| std::env::var("SDZ_AUTH_PROJECT_ID"))
        {
            let token = std::env::var("SDZ_FIRESTORE_TOKEN").ok();
            let on_cloud_run = std::env::var("K_SERVICE").is_ok();
            if token.is_some() || on_cloud_run {
                if let (Ok(user_repo), Ok(spot_repo), Ok(mylist_repo)) = (
                    SdzFirestoreUserRepository::new(project_id.clone(), token.clone()),
                    SdzFirestoreSpotRepository::new(project_id.clone(), token.clone()),
                    SdzFirestoreMyListRepository::new(project_id.clone(), token.clone()),
                ) {
                    return SdzAppState {
                        user_repo: Arc::new(user_repo),
                        spot_repo: Arc::new(spot_repo),
                        mylist_repo: Arc::new(mylist_repo),
                        storage_repo,
                    };
                } else {
                    tracing::warn!("Failed to init Firestore repo, falling back to in-memory");
                }
            } else {
                tracing::warn!(
                    "SDZ_USE_FIRESTORE=1 but token missing and not Cloud Run. Falling back to in-memory."
                );
            }
        } else {
            tracing::warn!(
                "SDZ_USE_FIRESTORE=1 but project_id missing. Falling back to in-memory."
            );
        }
    }

    // デフォルト: ローカル動作用のメモリリポジトリ
    let seed_user = SdzUser {
        sdz_user_id: "zjJuiae1ymc6kqjU88yFsJvAuxG2".to_string(),
        sdz_display_name: "sdz-demo-user".to_string(),
        sdz_email: Some("uechi@321dev.org".to_string()),
    };
    let repo = SdzInMemoryUserRepository::new_with_seed(vec![seed_user]);
    SdzAppState {
        user_repo: Arc::new(repo),
        spot_repo: Arc::new(SdzInMemorySpotRepository::default()),
        mylist_repo: Arc::new(SdzInMemoryMyListRepository::default()),
        storage_repo,
    }
}

#[derive(Clone)]
pub struct SdzAppState {
    pub user_repo: Arc<dyn SdzUserRepository>,
    pub spot_repo: Arc<dyn SdzSpotRepository>,
    pub mylist_repo: Arc<dyn SdzMyListRepository>,
    pub storage_repo: Arc<dyn SdzStorageRepository>,
}

fn build_storage_repo() -> Arc<dyn SdzStorageRepository> {
    let bucket = std::env::var("SDZ_STORAGE_BUCKET").ok();
    let service_account = std::env::var("SDZ_STORAGE_SERVICE_ACCOUNT_EMAIL").ok();
    let expires = std::env::var("SDZ_STORAGE_SIGNED_URL_EXPIRES_SECS")
        .ok()
        .and_then(|v| v.parse::<u32>().ok())
        .unwrap_or(900);

    if let (Some(bucket), Some(service_account)) = (bucket, service_account) {
        match SdzStorageSignedUrlRepository::new(bucket, service_account, expires) {
            Ok(repo) => return Arc::new(repo),
            Err(_) => tracing::warn!("Failed to init storage repo, falling back to disabled"),
        }
    } else {
        tracing::warn!("Storage config missing, upload-url will be disabled");
    }

    Arc::new(SdzDisabledStorageRepository)
}
