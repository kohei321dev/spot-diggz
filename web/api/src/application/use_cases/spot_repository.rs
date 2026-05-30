use async_trait::async_trait;

use crate::{domain::models::SdzSpot, presentation::error::SdzApiError};

#[allow(dead_code)]
#[async_trait]
pub trait SdzSpotRepository: Send + Sync {
    async fn create(&self, spot: SdzSpot) -> Result<SdzSpot, SdzApiError>;
    async fn update(&self, spot: SdzSpot) -> Result<SdzSpot, SdzApiError>;
    async fn find_by_id(&self, spot_id: &str) -> Result<Option<SdzSpot>, SdzApiError>;
    async fn list_recent(&self, limit: usize) -> Result<Vec<SdzSpot>, SdzApiError>;
    async fn count_image_spots_by_user(&self, user_id: &str) -> Result<usize, SdzApiError>;
}
