use std::sync::Arc;

use crate::{
    application::use_cases::spot_repository::SdzSpotRepository,
    domain::models::{
        SdzSpot, SdzSpotApprovalStatus, SdzSpotLocation, SdzSpotParkAttributes,
        SdzSpotValidationError, SdzStreetAttributes, SdzUpdateSpotParams,
    },
    presentation::error::SdzApiError,
};

pub struct SdzUpdateSpotUseCase;

impl SdzUpdateSpotUseCase {
    pub fn new() -> Self {
        Self
    }

    pub async fn execute(
        &self,
        repo: Arc<dyn SdzSpotRepository>,
        spot_id: String,
        input: UpdateSpotInput,
    ) -> Result<SdzSpot, SdzApiError> {
        let existing = repo
            .find_by_id(&spot_id)
            .await?
            .ok_or(SdzApiError::NotFound)?;

        let updated = existing
            .update(SdzUpdateSpotParams {
                name: input.name,
                description: input.description,
                location: input.location.map(|loc| SdzSpotLocation {
                    lat: loc.lat,
                    lng: loc.lng,
                }),
                tags: input.tags,
                images: input.images,
                sdz_approval_status: input.approval_status,
                sdz_park_attributes: input.park_attributes,
                sdz_street_attributes: input.street_attributes,
                sdz_instagram_tag: input.instagram_tag,
                sdz_instagram_location_url: input.instagram_location_url,
                sdz_instagram_profile_url: input.instagram_profile_url,
                sdz_google_place_id: input.google_place_id,
                sdz_google_maps_url: input.google_maps_url,
                sdz_address: input.address,
                sdz_phone_number: input.phone_number,
                sdz_google_rating: input.google_rating,
                sdz_google_rating_count: input.google_rating_count,
                sdz_google_types: input.google_types,
            })
            .map_err(map_validation_error)?;

        repo.update(updated.clone()).await?;
        Ok(updated)
    }
}

#[derive(Debug, Clone, serde::Deserialize)]
pub struct UpdateSpotInput {
    pub name: Option<String>,
    pub description: Option<String>,
    pub location: Option<UpdateSpotLocation>,
    pub tags: Option<Vec<String>>,
    pub images: Option<Vec<String>>,
    #[serde(rename = "approvalStatus")]
    pub approval_status: Option<SdzSpotApprovalStatus>,
    #[serde(rename = "parkAttributes")]
    pub park_attributes: Option<SdzSpotParkAttributes>,
    #[serde(rename = "streetAttributes")]
    pub street_attributes: Option<SdzStreetAttributes>,
    #[serde(rename = "instagramTag")]
    pub instagram_tag: Option<String>,
    #[serde(rename = "instagramLocationUrl")]
    pub instagram_location_url: Option<String>,
    #[serde(rename = "instagramProfileUrl")]
    pub instagram_profile_url: Option<String>,
    #[serde(rename = "googlePlaceId")]
    pub google_place_id: Option<String>,
    #[serde(rename = "googleMapsUrl")]
    pub google_maps_url: Option<String>,
    pub address: Option<String>,
    #[serde(rename = "phoneNumber")]
    pub phone_number: Option<String>,
    #[serde(rename = "googleRating")]
    pub google_rating: Option<f64>,
    #[serde(rename = "googleRatingCount")]
    pub google_rating_count: Option<u32>,
    #[serde(rename = "googleTypes")]
    pub google_types: Option<Vec<String>>,
}

#[derive(Debug, Clone, serde::Deserialize)]
pub struct UpdateSpotLocation {
    pub lat: f64,
    pub lng: f64,
}

fn map_validation_error(err: SdzSpotValidationError) -> SdzApiError {
    SdzApiError::BadRequest(err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::models::SdzCreateSpotParams;
    use crate::infrastructure::in_memory_spot_repository::SdzInMemorySpotRepository;

    async fn seed_spot(repo: &Arc<dyn SdzSpotRepository>) -> SdzSpot {
        let spot = SdzSpot::new_with_id(SdzCreateSpotParams {
            sdz_spot_id: "spot-1".into(),
            name: "test park".into(),
            description: Some("desc".into()),
            location: None,
            tags: vec!["park".into()],
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
        })
        .unwrap();
        repo.create(spot).await.unwrap()
    }

    #[tokio::test]
    async fn update_spot_name() {
        let repo: Arc<dyn SdzSpotRepository> = Arc::new(SdzInMemorySpotRepository::default());
        seed_spot(&repo).await;

        let use_case = SdzUpdateSpotUseCase::new();
        let input = UpdateSpotInput {
            name: Some("updated name".into()),
            description: None,
            location: None,
            tags: None,
            images: None,
            approval_status: None,
            park_attributes: None,
            street_attributes: None,
            instagram_tag: None,
            instagram_location_url: None,
            instagram_profile_url: None,
            google_place_id: None,
            google_maps_url: None,
            address: None,
            phone_number: None,
            google_rating: None,
            google_rating_count: None,
            google_types: None,
        };

        let result = use_case
            .execute(repo.clone(), "spot-1".into(), input)
            .await
            .unwrap();
        assert_eq!(result.name, "updated name");
        assert_eq!(result.description, Some("desc".into()));
    }

    #[tokio::test]
    async fn update_spot_not_found() {
        let repo: Arc<dyn SdzSpotRepository> = Arc::new(SdzInMemorySpotRepository::default());

        let use_case = SdzUpdateSpotUseCase::new();
        let input = UpdateSpotInput {
            name: Some("name".into()),
            description: None,
            location: None,
            tags: None,
            images: None,
            approval_status: None,
            park_attributes: None,
            street_attributes: None,
            instagram_tag: None,
            instagram_location_url: None,
            instagram_profile_url: None,
            google_place_id: None,
            google_maps_url: None,
            address: None,
            phone_number: None,
            google_rating: None,
            google_rating_count: None,
            google_types: None,
        };

        let err = use_case
            .execute(repo, "nonexistent".into(), input)
            .await
            .unwrap_err();
        assert!(matches!(err, SdzApiError::NotFound));
    }
}
