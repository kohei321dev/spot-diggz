use async_trait::async_trait;
use chrono::{DateTime, FixedOffset};
use reqwest::Client;
use serde::Deserialize;
use serde_json::json;

use crate::{
    application::use_cases::mylist_repository::SdzMyListRepository, domain::models::SdzMyListEntry,
    presentation::error::SdzApiError,
};

pub struct SdzFirestoreMyListRepository {
    project_id: String,
    bearer_token: Option<String>,
    http: Client,
}

impl SdzFirestoreMyListRepository {
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

    async fn upsert_document(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError> {
        let url = format!(
            "https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents/users/{}/mylist/{}",
            self.project_id, user_id, spot_id
        );

        let body = json!({
            "fields": {
                "spotId": { "stringValue": spot_id },
                "createdAt": { "timestampValue": now_jst_string() }
            }
        });
        let token = self.resolve_token().await?;

        let resp = self
            .http
            .patch(url)
            .bearer_auth(token)
            .json(&body)
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Firestore request error: {:?}", e);
                SdzApiError::Internal
            })?;

        map_status(resp.status(), resp.text().await).map(|_| ())
    }

    async fn delete_document(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError> {
        let url = format!(
            "https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents/users/{}/mylist/{}",
            self.project_id, user_id, spot_id
        );
        let token = self.resolve_token().await?;
        let resp = self
            .http
            .delete(url)
            .bearer_auth(token)
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Firestore request error: {:?}", e);
                SdzApiError::Internal
            })?;

        match resp.status() {
            reqwest::StatusCode::OK | reqwest::StatusCode::NOT_FOUND => Ok(()),
            code => {
                let body = resp.text().await.unwrap_or_default();
                tracing::error!("Firestore unexpected status: {} body: {}", code, body);
                Err(SdzApiError::Internal)
            }
        }
    }

    async fn list_documents(&self, user_id: &str) -> Result<Vec<FirestoreMyListDoc>, SdzApiError> {
        let url = format!(
            "https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents/users/{}/mylist",
            self.project_id, user_id
        );
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
                let list = resp.json::<FirestoreListResponse>().await.map_err(|e| {
                    tracing::error!("Failed to parse Firestore response: {:?}", e);
                    SdzApiError::Internal
                })?;
                Ok(list.documents.unwrap_or_default())
            }
            reqwest::StatusCode::NOT_FOUND => Ok(vec![]),
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
impl SdzMyListRepository for SdzFirestoreMyListRepository {
    async fn list_by_user(&self, user_id: &str) -> Result<Vec<SdzMyListEntry>, SdzApiError> {
        let docs = self.list_documents(user_id).await?;
        let mut entries = Vec::new();
        for doc in docs {
            if let Some(entry) = doc.to_entry() {
                entries.push(entry);
            }
        }
        Ok(entries)
    }

    async fn add(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError> {
        self.upsert_document(user_id, spot_id).await
    }

    async fn remove(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError> {
        self.delete_document(user_id, spot_id).await
    }
}

#[derive(Debug, Deserialize)]
struct FirestoreListResponse {
    documents: Option<Vec<FirestoreMyListDoc>>,
}

#[derive(Debug, Deserialize)]
struct FirestoreMyListDoc {
    name: String,
    fields: Option<FirestoreMyListFields>,
}

impl FirestoreMyListDoc {
    fn to_entry(&self) -> Option<SdzMyListEntry> {
        let spot_id = self
            .fields
            .as_ref()
            .and_then(|fields| fields.spot_id.as_ref().map(|f| f.string_value.clone()))
            .or_else(|| extract_doc_id(&self.name))?;
        let created_at = self
            .fields
            .as_ref()
            .and_then(|fields| fields.created_at.as_ref())
            .and_then(|field| DateTime::parse_from_rfc3339(&field.timestamp_value).ok())
            .unwrap_or_else(now_jst);
        Some(SdzMyListEntry {
            sdz_spot_id: spot_id,
            created_at,
        })
    }
}

#[derive(Debug, Deserialize)]
struct FirestoreMyListFields {
    #[serde(rename = "spotId")]
    spot_id: Option<StringField>,
    #[serde(rename = "createdAt")]
    created_at: Option<TimestampField>,
}

#[derive(Debug, Deserialize)]
struct StringField {
    #[serde(rename = "stringValue")]
    string_value: String,
}

#[derive(Debug, Deserialize)]
struct TimestampField {
    #[serde(rename = "timestampValue")]
    timestamp_value: String,
}

#[derive(Debug, Deserialize)]
struct SdzMetadataToken {
    #[serde(rename = "access_token")]
    access_token: String,
}

fn extract_doc_id(name: &str) -> Option<String> {
    name.split('/').next_back().map(|value| value.to_string())
}

fn now_jst_string() -> String {
    now_jst().to_rfc3339()
}

fn now_jst() -> DateTime<FixedOffset> {
    let offset = FixedOffset::east_opt(9 * 3600).expect("valid offset");
    chrono::Utc::now().with_timezone(&offset)
}

fn map_status(
    status: reqwest::StatusCode,
    body: Result<String, reqwest::Error>,
) -> Result<(), SdzApiError> {
    if status.is_success() {
        return Ok(());
    }
    let text = body.unwrap_or_default();
    tracing::error!("Firestore unexpected status: {} body: {}", status, text);
    Err(SdzApiError::Internal)
}
