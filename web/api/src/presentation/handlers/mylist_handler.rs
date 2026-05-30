use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use serde::Serialize;

use crate::{
    application::use_cases::{
        add_mylist_use_case::{SdzAddMyListInput, SdzAddMyListUseCase},
        list_mylist_use_case::SdzListMyListUseCase,
        remove_mylist_use_case::SdzRemoveMyListUseCase,
    },
    presentation::{error::SdzApiError, middleware::auth::SdzAuthUser, router::SdzAppState},
};

#[derive(Debug, Serialize)]
struct SdzMyListActionResponse {
    #[serde(rename = "spotId")]
    sdz_spot_id: String,
    status: String,
}

pub async fn handle_list_mylist(
    State(state): State<SdzAppState>,
    auth_user: SdzAuthUser,
) -> impl IntoResponse {
    let use_case = SdzListMyListUseCase::new();
    let spots = use_case
        .execute(
            state.mylist_repo.clone(),
            state.spot_repo.clone(),
            auth_user,
        )
        .await?;
    Ok::<_, SdzApiError>((StatusCode::OK, Json(spots)))
}

pub async fn handle_add_mylist(
    State(state): State<SdzAppState>,
    auth_user: SdzAuthUser,
    Json(payload): Json<SdzAddMyListInput>,
) -> impl IntoResponse {
    let spot_id = payload.sdz_spot_id.clone();
    let use_case = SdzAddMyListUseCase::new();
    use_case
        .execute(
            state.mylist_repo.clone(),
            state.spot_repo.clone(),
            auth_user,
            payload,
        )
        .await?;
    Ok::<_, SdzApiError>((
        StatusCode::OK,
        Json(SdzMyListActionResponse {
            sdz_spot_id: spot_id,
            status: "added".to_string(),
        }),
    ))
}

pub async fn handle_remove_mylist(
    State(state): State<SdzAppState>,
    Path(spot_id): Path<String>,
    auth_user: SdzAuthUser,
) -> impl IntoResponse {
    let use_case = SdzRemoveMyListUseCase::new();
    use_case
        .execute(state.mylist_repo.clone(), auth_user, spot_id.clone())
        .await?;
    Ok::<_, SdzApiError>((
        StatusCode::OK,
        Json(SdzMyListActionResponse {
            sdz_spot_id: spot_id,
            status: "removed".to_string(),
        }),
    ))
}
