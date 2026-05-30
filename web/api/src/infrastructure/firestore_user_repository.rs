use async_trait::async_trait;
use reqwest::Client;
use serde::Deserialize;

use crate::{
    application::use_cases::user_repository::SdzUserRepository, domain::models::SdzUser,
    presentation::error::SdzApiError,
};

/// Firestoreの`users`コレクションからユーザーを取得するリポジトリ。
/// 認証には環境変数`SDZ_FIRESTORE_TOKEN`かCloud Runのメタデータサーバートークンを使用する。
pub struct SdzFirestoreUserRepository {
    project_id: String,
    bearer_token: Option<String>,
    http: Client,
}

impl SdzFirestoreUserRepository {
    pub fn new(project_id: String, bearer_token: Option<String>) -> Result<Self, SdzApiError> {
        let http = Client::builder().build().map_err(|e| {
            tracing::error!("Failed to build reqwest client: {:?}", e);
            SdzApiError::Internal
        })?;
        Ok(Self {
            project_id,
            bearer_token,
            http,
        })
    }

    async fn get_document(&self, user_id: &str) -> Result<Option<FirestoreUserDoc>, SdzApiError> {
        let url = format!("https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents/users/{}", self.project_id, user_id);
        let token = self.resolve_token().await?;
        let resp = self
            .http
            .get(url)
            .bearer_auth(token)
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Firestore request error: {:?}", e);
                SdzApiError::Internal
            })?;

        match resp.status() {
            reqwest::StatusCode::OK => {
                let doc = resp.json::<FirestoreUserDoc>().await.map_err(|e| {
                    tracing::error!("Failed to parse Firestore response: {:?}", e);
                    SdzApiError::Internal
                })?;
                Ok(Some(doc))
            }
            reqwest::StatusCode::NOT_FOUND => Ok(None),
            code => {
                let body = resp.text().await.unwrap_or_default();
                tracing::error!("Firestore unexpected status: {} body: {}", code, body);
                Err(SdzApiError::Internal)
            }
        }
    }

    async fn resolve_token(&self) -> Result<String, SdzApiError> {
        if let Some(token) = self
            .bearer_token
            .as_ref()
            .filter(|token| !token.trim().is_empty())
        {
            return Ok(token.to_string());
        }
        if let Ok(token) = std::env::var("SDZ_FIRESTORE_TOKEN") {
            if !token.trim().is_empty() {
                return Ok(token);
            }
        }
        self.fetch_metadata_token().await
    }

    async fn fetch_metadata_token(&self) -> Result<String, SdzApiError> {
        let metadata_url = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token";
        let resp = self
            .http
            .get(metadata_url)
            .header("Metadata-Flavor", "Google")
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Failed to fetch metadata token: {:?}", e);
                SdzApiError::Internal
            })?;

        if !resp.status().is_success() {
            let body = resp.text().await.unwrap_or_default();
            tracing::error!("Metadata token error: {}", body);
            return Err(SdzApiError::Internal);
        }

        let token = resp.json::<SdzMetadataToken>().await.map_err(|e| {
            tracing::error!("Failed to parse metadata token: {:?}", e);
            SdzApiError::Internal
        })?;

        Ok(token.access_token)
    }
}

#[async_trait]
impl SdzUserRepository for SdzFirestoreUserRepository {
    async fn find_by_id(&self, user_id: &str) -> Option<SdzUser> {
        match self.get_document(user_id).await {
            Ok(Some(doc)) => Some(SdzUser {
                sdz_user_id: user_id.to_string(),
                sdz_display_name: doc
                    .fields
                    .display_name
                    .map(|f| f.string_value)
                    .unwrap_or_else(|| "unknown".to_string()),
                sdz_email: doc.fields.email.map(|f| f.string_value),
            }),
            Ok(None) => None,
            Err(_) => None,
        }
    }
}

// Firestore RESTのレスポンスモデル（必要最小限のみ）
#[derive(Debug, Deserialize)]
struct FirestoreUserDoc {
    fields: FirestoreUserFields,
}

#[derive(Debug, Deserialize)]
struct FirestoreUserFields {
    #[serde(rename = "displayName")]
    display_name: Option<StringField>,
    email: Option<StringField>,
}

#[derive(Debug, Deserialize)]
struct StringField {
    #[serde(rename = "stringValue")]
    string_value: String,
}

#[derive(Debug, Deserialize)]
struct SdzMetadataToken {
    #[serde(rename = "access_token")]
    access_token: String,
}
