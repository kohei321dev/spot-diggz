use std::collections::HashMap;
use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, FixedOffset};
use tokio::sync::RwLock;

use crate::{
    application::use_cases::mylist_repository::SdzMyListRepository, domain::models::SdzMyListEntry,
    presentation::error::SdzApiError,
};

#[derive(Clone, Default)]
pub struct SdzInMemoryMyListRepository {
    store: Arc<RwLock<HashMap<String, Vec<SdzMyListEntry>>>>,
}

#[async_trait]
impl SdzMyListRepository for SdzInMemoryMyListRepository {
    async fn list_by_user(&self, user_id: &str) -> Result<Vec<SdzMyListEntry>, SdzApiError> {
        let store = self.store.read().await;
        Ok(store.get(user_id).cloned().unwrap_or_default())
    }

    async fn add(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError> {
        let mut store = self.store.write().await;
        let list = store.entry(user_id.to_string()).or_default();
        if list.iter().any(|item| item.sdz_spot_id == spot_id) {
            return Ok(());
        }
        list.insert(
            0,
            SdzMyListEntry {
                sdz_spot_id: spot_id.to_string(),
                created_at: now_jst(),
            },
        );
        Ok(())
    }

    async fn remove(&self, user_id: &str, spot_id: &str) -> Result<(), SdzApiError> {
        let mut store = self.store.write().await;
        if let Some(list) = store.get_mut(user_id) {
            list.retain(|item| item.sdz_spot_id != spot_id);
        }
        Ok(())
    }
}

fn now_jst() -> DateTime<FixedOffset> {
    let offset = FixedOffset::east_opt(9 * 3600).expect("valid offset");
    chrono::Utc::now().with_timezone(&offset)
}
