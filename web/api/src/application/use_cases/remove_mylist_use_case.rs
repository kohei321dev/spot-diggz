use std::sync::Arc;

use crate::{
    application::use_cases::mylist_repository::SdzMyListRepository,
    presentation::{error::SdzApiError, middleware::auth::SdzAuthUser},
};

pub struct SdzRemoveMyListUseCase;

impl SdzRemoveMyListUseCase {
    pub fn new() -> Self {
        Self
    }

    pub async fn execute(
        &self,
        mylist_repo: Arc<dyn SdzMyListRepository>,
        auth_user: SdzAuthUser,
        spot_id: String,
    ) -> Result<(), SdzApiError> {
        mylist_repo.remove(&auth_user.sdz_user_id, &spot_id).await?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::infrastructure::in_memory_mylist_repository::SdzInMemoryMyListRepository;

    #[tokio::test]
    async fn remove_mylist_clears_entry() {
        let mylist_repo = Arc::new(SdzInMemoryMyListRepository::default());
        mylist_repo.add("user-1", "spot-1").await.unwrap();

        let use_case = SdzRemoveMyListUseCase::new();
        let auth = SdzAuthUser {
            sdz_user_id: "user-1".to_string(),
        };
        use_case
            .execute(mylist_repo.clone(), auth, "spot-1".to_string())
            .await
            .unwrap();

        let list = mylist_repo.list_by_user("user-1").await.unwrap();
        assert!(list.is_empty());
    }
}
