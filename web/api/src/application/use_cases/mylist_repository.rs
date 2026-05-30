use async_trait::async_trait;

use crate::domain::models::SdzMyListEntry;
use crate::presentation::error::SdzApiError;

#[async_trait]
pub trait SdzMyListRepository: Send + Sync {
    async fn list_by_user(&self, user_id: &str) -> Result<Vec<SdzMyListEntry>, SdzApiError>;
    async fn add(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError>;
    async fn remove(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError>;
}
