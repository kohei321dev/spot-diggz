use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::RwLock;

use crate::{
    application::use_cases::spot_repository::SdzSpotRepository, domain::models::SdzSpot,
    presentation::error::SdzApiError,
};

#[derive(Clone, Default)]
pub struct SdzInMemorySpotRepository {
    store: Arc<RwLock<HashMap<String, SdzSpot>>>,
}

#[async_trait]
impl SdzSpotRepository for SdzInMemorySpotRepository {
    async fn create(&self, spot: SdzSpot) -> Result<SdzSpot, SdzApiError> {
        let mut store = self.store.write().await;
        store.insert(spot.sdz_spot_id.clone(), spot.clone());
        Ok(spot)
    }

    async fn update(&self, spot: SdzSpot) -> Result<SdzSpot, SdzApiError> {
        let mut store = self.store.write().await;
        if !store.contains_key(&spot.sdz_spot_id) {
            return Err(SdzApiError::NotFound);
        }
        store.insert(spot.sdz_spot_id.clone(), spot.clone());
        Ok(spot)
    }

    async fn find_by_id(&self, spot_id: &str) -> Result<Option<SdzSpot>, SdzApiError> {
        let store = self.store.read().await;
        Ok(store.get(spot_id).cloned())
    }

    async fn list_recent(&self, limit: usize) -> Result<Vec<SdzSpot>, SdzApiError> {
        let store = self.store.read().await;
        let mut list: Vec<_> = store.values().cloned().collect();
        list.sort_by(|a, b| b.created_at.cmp(&a.created_at));
        list.truncate(limit);
        Ok(list)
    }

    async fn count_image_spots_by_user(&self, user_id: &str) -> Result<usize, SdzApiError> {
        let store = self.store.read().await;
        let count = store
            .values()
            .filter(|spot| spot.sdz_user_id == user_id && !spot.images.is_empty())
            .count();
        Ok(count)
    }
}
