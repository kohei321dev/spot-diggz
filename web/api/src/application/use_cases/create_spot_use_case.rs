use std::sync::Arc;

use uuid::Uuid;

use crate::{
    application::use_cases::spot_repository::SdzSpotRepository,
    domain::models::{
        SdzCreateSpotParams, SdzSpot, SdzSpotApprovalStatus, SdzSpotLocation,
        SdzSpotParkAttributes, SdzSpotValidationError, SdzStreetAttributes,
    },
    presentation::error::SdzApiError,
    presentation::middleware::auth::SdzAuthUser,
};

pub struct SdzCreateSpotUseCase;

impl SdzCreateSpotUseCase {
    pub fn new() -> Self {
        Self
    }

    pub async fn execute(
        &self,
        repo: Arc<dyn SdzSpotRepository>,
        auth_user: SdzAuthUser,
        input: CreateSpotInput,
    ) -> Result<SdzSpot, SdzApiError> {
        let spot = SdzSpot::new_with_id(SdzCreateSpotParams {
            sdz_spot_id: Uuid::new_v4().to_string(),
            name: input.name,
            description: input.description,
            location: input.location.map(|loc| SdzSpotLocation {
                lat: loc.lat,
                lng: loc.lng,
            }),
            tags: input.tags.unwrap_or_default(),
            images: input.images.unwrap_or_default(),
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
            sdz_google_types: input.google_types.unwrap_or_default(),
            sdz_user_id: auth_user.sdz_user_id,
        })
        .map_err(map_validation_error)?;

        repo.create(spot.clone()).await?;

        Ok(spot)
    }
}

#[derive(Debug, Clone, serde::Deserialize)]
pub struct CreateSpotInput {
    pub name: String,
    pub description: Option<String>,
    pub location: Option<CreateSpotLocation>,
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
pub struct CreateSpotLocation {
    pub lat: f64,
    pub lng: f64,
}

fn map_validation_error(err: SdzSpotValidationError) -> SdzApiError {
    SdzApiError::BadRequest(err.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::infrastructure::in_memory_spot_repository::SdzInMemorySpotRepository;

    fn build_input() -> CreateSpotInput {
        CreateSpotInput {
            name: "test spot".into(),
            description: Some("desc".into()),
            location: Some(CreateSpotLocation {
                lat: 35.0,
                lng: 139.0,
            }),
            tags: Some(vec!["park".into()]),
            images: Some(vec![]),
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
        }
    }

    #[tokio::test]
    async fn create_spot_success() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let auth = SdzAuthUser {
            sdz_user_id: "user-1".into(),
        };
        let input = build_input();
        let use_case = SdzCreateSpotUseCase::new();

        let result = use_case.execute(repo.clone(), auth, input).await.unwrap();

        assert_eq!(result.name, "test spot");
        assert_eq!(result.sdz_user_id, "user-1");
        assert!(!result.sdz_spot_id.is_empty());
        assert_eq!(result.tags.len(), 1);
        assert!(result.sdz_approval_status.is_none());
        assert!(repo
            .find_by_id(&result.sdz_spot_id)
            .await
            .unwrap()
            .is_some());
    }

    #[tokio::test]
    async fn create_spot_invalid_lat() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let auth = SdzAuthUser {
            sdz_user_id: "user-1".into(),
        };
        let mut input = build_input();
        input.location = Some(CreateSpotLocation {
            lat: 100.0,
            lng: 139.0,
        });
        let use_case = SdzCreateSpotUseCase::new();

        let err = use_case.execute(repo, auth, input).await.unwrap_err();
        match err {
            SdzApiError::BadRequest(msg) => assert!(msg.contains("lat")),
            _ => panic!("expected bad request"),
        }
    }
}
