use std::sync::Arc;

use crate::{
    application::use_cases::spot_repository::SdzSpotRepository, domain::models::SdzSpot,
    presentation::error::SdzApiError,
};

pub struct SdzGetSpotUseCase;

impl SdzGetSpotUseCase {
    pub fn new() -> Self {
        Self
    }

    pub async fn execute(
        &self,
        repo: Arc<dyn SdzSpotRepository>,
        spot_id: String,
        viewer_user_id: Option<String>,
    ) -> Result<SdzSpot, SdzApiError> {
        let spot = repo
            .find_by_id(&spot_id)
            .await?
            .ok_or(SdzApiError::NotFound)?;

        if spot.is_approved() {
            return Ok(spot);
        }

        if let Some(user_id) = viewer_user_id {
            if spot.sdz_user_id == user_id {
                return Ok(spot);
            }
        }

        Err(SdzApiError::NotFound)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        domain::models::SdzSpot,
        infrastructure::in_memory_spot_repository::SdzInMemorySpotRepository,
    };
    use chrono::{FixedOffset, TimeZone};

    #[tokio::test]
    async fn get_spot_not_found() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let use_case = SdzGetSpotUseCase::new();
        let err = use_case
            .execute(repo, "missing".into(), None)
            .await
            .unwrap_err();
        matches!(err, SdzApiError::NotFound);
    }

    #[tokio::test]
    async fn get_spot_requires_approval_for_public() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let tz = FixedOffset::east_opt(9 * 3600).unwrap();
        repo.create(SdzSpot {
            sdz_spot_id: "spot-1".into(),
            name: "A".into(),
            description: None,
            location: None,
            tags: vec![],
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
            sdz_user_id: "user-1".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();

        let use_case = SdzGetSpotUseCase::new();
        let err = use_case
            .execute(repo.clone(), "spot-1".into(), None)
            .await
            .unwrap_err();
        match err {
            SdzApiError::NotFound => {}
            _ => panic!("expected not found"),
        }

        let spot = use_case
            .execute(repo, "spot-1".into(), Some("user-1".into()))
            .await
            .unwrap();
        assert_eq!(spot.sdz_spot_id, "spot-1");
    }
}
