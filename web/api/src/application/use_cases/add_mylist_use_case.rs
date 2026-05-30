use std::sync::Arc;

use crate::{
    application::use_cases::{
        mylist_repository::SdzMyListRepository, spot_repository::SdzSpotRepository,
    },
    presentation::{error::SdzApiError, middleware::auth::SdzAuthUser},
};

pub struct SdzAddMyListUseCase;

impl SdzAddMyListUseCase {
    pub fn new() -> Self {
        Self
    }

    pub async fn execute(
        &self,
        mylist_repo: Arc<dyn SdzMyListRepository>,
        spot_repo: Arc<dyn SdzSpotRepository>,
        auth_user: SdzAuthUser,
        input: SdzAddMyListInput,
    ) -> Result<(), SdzApiError> {
        let existing = spot_repo.find_by_id(&input.sdz_spot_id).await?;
        if existing.is_none() {
            return Err(SdzApiError::NotFound);
        }
        mylist_repo
            .add(&auth_user.sdz_user_id, &input.sdz_spot_id)
            .await?;
        Ok(())
    }
}

#[derive(Debug, Clone, serde::Deserialize)]
pub struct SdzAddMyListInput {
    #[serde(rename = "spotId")]
    pub sdz_spot_id: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        domain::models::{SdzCreateSpotParams, SdzSpot, SdzSpotLocation},
        infrastructure::{
            in_memory_mylist_repository::SdzInMemoryMyListRepository,
            in_memory_spot_repository::SdzInMemorySpotRepository,
        },
    };

    fn sample_spot(id: &str, user_id: &str) -> SdzSpot {
        SdzSpot::new_with_id(SdzCreateSpotParams {
            sdz_spot_id: id.to_string(),
            name: "sample".to_string(),
            description: Some("desc".to_string()),
            location: Some(SdzSpotLocation {
                lat: 35.0,
                lng: 139.0,
            }),
            tags: vec!["park".to_string()],
            images: vec![],
            sdz_approval_status: None,
            sdz_park_attributes: None,
            sdz_street_attributes: None,
            sdz_instagram_tag: None,
            sdz_instagram_location_url: None,
            sdz_instagram_profile_url: None,
            sdz_google_place_id: None,
            sdz_google_maps_url: None,
            sdz_address: None,
            sdz_phone_number: None,
            sdz_google_rating: None,
            sdz_google_rating_count: None,
            sdz_google_types: vec![],
            sdz_user_id: user_id.to_string(),
        })
        .expect("valid spot")
    }

    #[tokio::test]
    async fn add_mylist_requires_existing_spot() {
        let mylist_repo = Arc::new(SdzInMemoryMyListRepository::default());
        let spot_repo = Arc::new(SdzInMemorySpotRepository::default());
        let use_case = SdzAddMyListUseCase::new();
        let auth = SdzAuthUser {
            sdz_user_id: "user-1".to_string(),
        };
        let input = SdzAddMyListInput {
            sdz_spot_id: "missing".to_string(),
        };

        let err = use_case
            .execute(mylist_repo, spot_repo, auth, input)
            .await
            .unwrap_err();
        assert!(matches!(err, SdzApiError::NotFound));
    }

    #[tokio::test]
    async fn add_mylist_success() {
        let mylist_repo = Arc::new(SdzInMemoryMyListRepository::default());
        let spot_repo = Arc::new(SdzInMemorySpotRepository::default());
        spot_repo
            .create(sample_spot("spot-1", "user-1"))
            .await
            .unwrap();

        let use_case = SdzAddMyListUseCase::new();
        let auth = SdzAuthUser {
            sdz_user_id: "user-1".to_string(),
        };
        let input = SdzAddMyListInput {
            sdz_spot_id: "spot-1".to_string(),
        };

        use_case
            .execute(mylist_repo.clone(), spot_repo, auth, input)
            .await
            .unwrap();

        let list = mylist_repo.list_by_user("user-1").await.unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].sdz_spot_id, "spot-1");
    }
}
