use axum::{
    extract::FromRequestParts,
    http::{header::AUTHORIZATION, request::Parts},
};
use axum_extra::{
    headers::{authorization::Bearer, Authorization},
    TypedHeader,
};
use jsonwebtoken::{decode, decode_header, Algorithm, DecodingKey, Validation};
use once_cell::sync::Lazy;
use reqwest::Client;
use serde::Deserialize;
use std::collections::HashMap;

use crate::presentation::error::SdzApiError;

/// 認証済みユーザー情報（今後Firebase JWT検証に差し替え予定）
#[derive(Debug, Clone)]
pub struct SdzAuthUser {
    pub sdz_user_id: String,
}

impl<S> FromRequestParts<S> for SdzAuthUser
where
    S: Send + Sync,
{
    type Rejection = SdzApiError;

    #[allow(clippy::manual_async_fn)]
    fn from_request_parts(
        parts: &mut Parts,
        state: &S,
    ) -> impl std::future::Future<Output = Result<Self, Self::Rejection>> + Send {
        async move {
            let TypedHeader(Authorization(bearer)) =
                TypedHeader::<Authorization<Bearer>>::from_request_parts(parts, state)
                    .await
                    .map_err(|_| SdzApiError::Unauthorized)?;

            let token = bearer.token();
            let claims = verify_firebase_token(token).await.map_err(|e| {
                tracing::error!("JWT verification failed: {:?}", e);
                SdzApiError::Unauthorized
            })?;

            Ok(SdzAuthUser {
                sdz_user_id: claims.sdz_user_id(),
            })
        }
    }
}

/// 任意の認証ユーザー情報（Authorizationが無ければNone）
#[allow(dead_code)]
#[derive(Debug, Clone)]
pub struct SdzOptionalAuthUser {
    pub sdz_user_id: Option<String>,
}

impl<S> FromRequestParts<S> for SdzOptionalAuthUser
where
    S: Send + Sync,
{
    type Rejection = SdzApiError;

    #[allow(clippy::manual_async_fn)]
    fn from_request_parts(
        parts: &mut Parts,
        _state: &S,
    ) -> impl std::future::Future<Output = Result<Self, Self::Rejection>> + Send {
        async move {
            let Some(header_value) = parts.headers.get(AUTHORIZATION) else {
                return Ok(SdzOptionalAuthUser { sdz_user_id: None });
            };
            let header_value = header_value
                .to_str()
                .map_err(|_| SdzApiError::Unauthorized)?;
            let token = header_value
                .strip_prefix("Bearer ")
                .ok_or(SdzApiError::Unauthorized)?;
            let claims = verify_firebase_token(token).await.map_err(|e| {
                tracing::error!("JWT verification failed: {:?}", e);
                SdzApiError::Unauthorized
            })?;
            Ok(SdzOptionalAuthUser {
                sdz_user_id: Some(claims.sdz_user_id()),
            })
        }
    }
}

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct FirebaseClaims {
    #[serde(default)]
    sub: String,
    #[serde(default)]
    user_id: String,
    #[serde(default)]
    aud: String,
    #[serde(default)]
    iss: String,
    #[serde(default)]
    exp: u64,
    #[serde(default)]
    iat: Option<u64>,
}

impl FirebaseClaims {
    fn sdz_user_id(&self) -> String {
        if !self.user_id.is_empty() {
            self.user_id.clone()
        } else {
            self.sub.clone()
        }
    }
}

static HTTP_CLIENT: Lazy<Client> = Lazy::new(|| Client::builder().build().expect("client"));

async fn fetch_jwks() -> Result<HashMap<String, String>, SdzApiError> {
    // Firebase/Identity Platformの公開鍵（kid->x509 cert）エンドポイント
    // 参考: https://firebase.google.com/docs/auth/admin/verify-id-tokens#retrieve_public_keys
    let jwks_url =
        "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com";
    let resp = HTTP_CLIENT
        .get(jwks_url)
        .send()
        .await
        .map_err(|e| {
            tracing::error!("Failed to fetch JWKS: {:?}", e);
            SdzApiError::Internal
        })?
        .json::<HashMap<String, String>>()
        .await
        .map_err(|e| {
            tracing::error!("Failed to parse JWKS: {:?}", e);
            SdzApiError::Internal
        })?;
    Ok(resp)
}

async fn verify_firebase_token(token: &str) -> Result<FirebaseClaims, SdzApiError> {
    let header = decode_header(token).map_err(|e| {
        tracing::error!("Failed to decode JWT header: {:?}", e);
        SdzApiError::Unauthorized
    })?;
    let kid = header.kid.ok_or_else(|| {
        tracing::error!("JWT header missing kid");
        SdzApiError::Unauthorized
    })?;

    let jwks = fetch_jwks().await?;
    let pem = jwks.get(&kid).ok_or_else(|| {
        tracing::error!("kid not found in JWKS: {}", kid);
        SdzApiError::Unauthorized
    })?;
    let decoding_key = DecodingKey::from_rsa_pem(pem.as_bytes()).map_err(|e| {
        tracing::error!("Failed to build decoding key: {:?}", e);
        SdzApiError::Unauthorized
    })?;

    let project_id = std::env::var("SDZ_AUTH_PROJECT_ID").map_err(|e| {
        tracing::error!("SDZ_AUTH_PROJECT_ID not set: {:?}", e);
        SdzApiError::Internal
    })?;
    let issuer = format!("https://securetoken.google.com/{}", project_id);

    let mut validation = Validation::new(Algorithm::RS256);
    validation.set_required_spec_claims(&["exp", "iss", "aud"]);
    validation.set_issuer(&[issuer]);
    validation.set_audience(&[project_id]);
    validation.validate_exp = true;

    let data = decode::<FirebaseClaims>(token, &decoding_key, &validation).map_err(|e| {
        tracing::error!("JWT decode/validate error: {:?}", e);
        SdzApiError::Unauthorized
    })?;

    Ok(data.claims)
}
