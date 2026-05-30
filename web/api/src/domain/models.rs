#[allow(dead_code)]
pub enum HealthStatus {
    Healthy,
    Degraded,
}

impl HealthStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Healthy => "healthy",
            Self::Degraded => "degraded",
        }
    }
}

use chrono::{DateTime, FixedOffset};
use serde::Deserialize;
use serde::Serialize;
use thiserror::Error;

#[derive(Debug, Clone, Serialize)]
pub struct SdzUser {
    #[serde(rename = "userId")]
    pub sdz_user_id: String,
    #[serde(rename = "displayName")]
    pub sdz_display_name: String,
    #[serde(rename = "email")]
    pub sdz_email: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzSpotLocation {
    pub lat: f64,
    pub lng: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzSpotTimeRange {
    #[serde(rename = "startMinutes")]
    pub start_minutes: u16,
    #[serde(rename = "endMinutes")]
    pub end_minutes: u16,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum SdzSpotBusinessScheduleType {
    Regular,
    WeekdayOnly,
    WeekendOnly,
    Irregular,
    SchoolOnly,
    Manual,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzSpotBusinessHours {
    #[serde(rename = "scheduleType", skip_serializing_if = "Option::is_none")]
    pub schedule_type: Option<SdzSpotBusinessScheduleType>,
    #[serde(rename = "is24Hours")]
    pub is_24_hours: bool,
    #[serde(rename = "sameAsWeekday")]
    pub same_as_weekday: bool,
    pub weekday: Option<SdzSpotTimeRange>,
    pub weekend: Option<SdzSpotTimeRange>,
    pub note: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzSpotParkAttributes {
    #[serde(rename = "officialUrl")]
    pub official_url: Option<String>,
    #[serde(rename = "businessHours")]
    pub business_hours: Option<SdzSpotBusinessHours>,
    #[serde(rename = "accessInfo")]
    pub access_info: Option<String>,
    #[serde(rename = "phoneNumber")]
    pub phone_number: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzStreetSurfaceCondition {
    pub roughness: Option<String>,
    pub crack: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzStreetSection {
    #[serde(rename = "type")]
    pub section_type: String,
    pub count: Option<u16>,
    #[serde(rename = "heightCm")]
    pub height_cm: Option<u16>,
    #[serde(rename = "widthCm")]
    pub width_cm: Option<u16>,
    pub notes: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzStreetAttributes {
    #[serde(rename = "surfaceMaterial")]
    pub surface_material: Option<String>,
    #[serde(rename = "surfaceCondition")]
    pub surface_condition: Option<SdzStreetSurfaceCondition>,
    pub sections: Option<Vec<SdzStreetSection>>,
    pub difficulty: Option<String>,
    pub notes: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SdzSpot {
    #[serde(rename = "spotId")]
    pub sdz_spot_id: String,
    pub name: String,
    pub description: Option<String>,
    pub location: Option<SdzSpotLocation>,
    pub tags: Vec<String>,
    pub images: Vec<String>,
    #[serde(rename = "approvalStatus", skip_serializing_if = "Option::is_none")]
    pub sdz_approval_status: Option<SdzSpotApprovalStatus>,
    #[serde(rename = "parkAttributes", skip_serializing_if = "Option::is_none")]
    pub sdz_park_attributes: Option<SdzSpotParkAttributes>,
    #[serde(rename = "streetAttributes", skip_serializing_if = "Option::is_none")]
    pub sdz_street_attributes: Option<SdzStreetAttributes>,
    #[serde(rename = "instagramTag", skip_serializing_if = "Option::is_none")]
    pub sdz_instagram_tag: Option<String>,
    #[serde(
        rename = "instagramLocationUrl",
        skip_serializing_if = "Option::is_none"
    )]
    pub sdz_instagram_location_url: Option<String>,
    #[serde(
        rename = "instagramProfileUrl",
        skip_serializing_if = "Option::is_none"
    )]
    pub sdz_instagram_profile_url: Option<String>,
    #[serde(rename = "googlePlaceId", skip_serializing_if = "Option::is_none")]
    pub sdz_google_place_id: Option<String>,
    #[serde(rename = "googleMapsUrl", skip_serializing_if = "Option::is_none")]
    pub sdz_google_maps_url: Option<String>,
    #[serde(rename = "address", skip_serializing_if = "Option::is_none")]
    pub sdz_address: Option<String>,
    #[serde(rename = "phoneNumber", skip_serializing_if = "Option::is_none")]
    pub sdz_phone_number: Option<String>,
    #[serde(rename = "googleRating", skip_serializing_if = "Option::is_none")]
    pub sdz_google_rating: Option<f64>,
    #[serde(rename = "googleRatingCount", skip_serializing_if = "Option::is_none")]
    pub sdz_google_rating_count: Option<u32>,
    #[serde(rename = "googleTypes", default, skip_serializing_if = "Vec::is_empty")]
    pub sdz_google_types: Vec<String>,
    #[serde(rename = "userId")]
    pub sdz_user_id: String,
    #[serde(rename = "createdAt")]
    pub created_at: DateTime<FixedOffset>,
    #[serde(rename = "updatedAt")]
    pub updated_at: DateTime<FixedOffset>,
}

#[derive(Debug, Clone)]
pub struct SdzMyListEntry {
    pub sdz_spot_id: String,
    pub created_at: DateTime<FixedOffset>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum SdzSpotApprovalStatus {
    Pending,
    Approved,
    Rejected,
}

/// スポット作成用パラメータ
pub struct SdzCreateSpotParams {
    pub sdz_spot_id: String,
    pub name: String,
    pub description: Option<String>,
    pub location: Option<SdzSpotLocation>,
    pub tags: Vec<String>,
    pub images: Vec<String>,
    pub sdz_approval_status: Option<SdzSpotApprovalStatus>,
    pub sdz_park_attributes: Option<SdzSpotParkAttributes>,
    pub sdz_street_attributes: Option<SdzStreetAttributes>,
    pub sdz_instagram_tag: Option<String>,
    pub sdz_instagram_location_url: Option<String>,
    pub sdz_instagram_profile_url: Option<String>,
    pub sdz_google_place_id: Option<String>,
    pub sdz_google_maps_url: Option<String>,
    pub sdz_address: Option<String>,
    pub sdz_phone_number: Option<String>,
    pub sdz_google_rating: Option<f64>,
    pub sdz_google_rating_count: Option<u32>,
    pub sdz_google_types: Vec<String>,
    pub sdz_user_id: String,
}

/// スポット更新用パラメータ（全フィールド Option で部分更新対応）
#[derive(Debug, Clone, Default)]
pub struct SdzUpdateSpotParams {
    pub name: Option<String>,
    pub description: Option<String>,
    pub location: Option<SdzSpotLocation>,
    pub tags: Option<Vec<String>>,
    pub images: Option<Vec<String>>,
    pub sdz_approval_status: Option<SdzSpotApprovalStatus>,
    pub sdz_park_attributes: Option<SdzSpotParkAttributes>,
    pub sdz_street_attributes: Option<SdzStreetAttributes>,
    pub sdz_instagram_tag: Option<String>,
    pub sdz_instagram_location_url: Option<String>,
    pub sdz_instagram_profile_url: Option<String>,
    pub sdz_google_place_id: Option<String>,
    pub sdz_google_maps_url: Option<String>,
    pub sdz_address: Option<String>,
    pub sdz_phone_number: Option<String>,
    pub sdz_google_rating: Option<f64>,
    pub sdz_google_rating_count: Option<u32>,
    pub sdz_google_types: Option<Vec<String>>,
}

impl SdzSpot {
    pub fn new_with_id(params: SdzCreateSpotParams) -> Result<Self, SdzSpotValidationError> {
        sdz_validate_spot(
            &params.name,
            params.location.as_ref(),
            &params.tags,
            &params.images,
            params.sdz_park_attributes.as_ref(),
            params.sdz_street_attributes.as_ref(),
        )?;
        if let Some(rating) = params.sdz_google_rating {
            sdz_validate_google_rating(rating)?;
        }
        Ok(Self {
            sdz_spot_id: params.sdz_spot_id,
            name: params.name,
            description: params.description,
            location: params.location,
            tags: params.tags,
            images: params.images,
            sdz_approval_status: params.sdz_approval_status,
            sdz_park_attributes: params.sdz_park_attributes,
            sdz_street_attributes: params.sdz_street_attributes,
            sdz_instagram_tag: params.sdz_instagram_tag,
            sdz_instagram_location_url: params.sdz_instagram_location_url,
            sdz_instagram_profile_url: params.sdz_instagram_profile_url,
            sdz_google_place_id: params.sdz_google_place_id,
            sdz_google_maps_url: params.sdz_google_maps_url,
            sdz_address: params.sdz_address,
            sdz_phone_number: params.sdz_phone_number,
            sdz_google_rating: params.sdz_google_rating,
            sdz_google_rating_count: params.sdz_google_rating_count,
            sdz_google_types: params.sdz_google_types,
            sdz_user_id: params.sdz_user_id,
            created_at: sdz_now_jst(),
            updated_at: sdz_now_jst(),
        })
    }

    pub fn is_approved(&self) -> bool {
        matches!(
            self.sdz_approval_status,
            Some(SdzSpotApprovalStatus::Approved)
        )
    }

    pub fn update(&self, params: SdzUpdateSpotParams) -> Result<Self, SdzSpotValidationError> {
        let name = params.name.unwrap_or_else(|| self.name.clone());
        let description = params.description.or_else(|| self.description.clone());
        let location = params.location.or_else(|| self.location.clone());
        let tags = params.tags.unwrap_or_else(|| self.tags.clone());
        let images = params.images.unwrap_or_else(|| self.images.clone());
        let approval_status = params
            .sdz_approval_status
            .or_else(|| self.sdz_approval_status.clone());
        let park_attributes = params
            .sdz_park_attributes
            .or_else(|| self.sdz_park_attributes.clone());
        let street_attributes = params
            .sdz_street_attributes
            .or_else(|| self.sdz_street_attributes.clone());
        let instagram_tag = params
            .sdz_instagram_tag
            .or_else(|| self.sdz_instagram_tag.clone());
        let instagram_location_url = params
            .sdz_instagram_location_url
            .or_else(|| self.sdz_instagram_location_url.clone());
        let instagram_profile_url = params
            .sdz_instagram_profile_url
            .or_else(|| self.sdz_instagram_profile_url.clone());
        let google_place_id = params
            .sdz_google_place_id
            .or_else(|| self.sdz_google_place_id.clone());
        let google_maps_url = params
            .sdz_google_maps_url
            .or_else(|| self.sdz_google_maps_url.clone());
        let address = params.sdz_address.or_else(|| self.sdz_address.clone());
        let phone_number = params
            .sdz_phone_number
            .or_else(|| self.sdz_phone_number.clone());
        let google_rating = params.sdz_google_rating.or(self.sdz_google_rating);
        let google_rating_count = params
            .sdz_google_rating_count
            .or(self.sdz_google_rating_count);
        let google_types = params
            .sdz_google_types
            .unwrap_or_else(|| self.sdz_google_types.clone());

        sdz_validate_spot(
            &name,
            location.as_ref(),
            &tags,
            &images,
            park_attributes.as_ref(),
            street_attributes.as_ref(),
        )?;
        if let Some(rating) = google_rating {
            sdz_validate_google_rating(rating)?;
        }

        Ok(Self {
            sdz_spot_id: self.sdz_spot_id.clone(),
            name,
            description,
            location,
            tags,
            images,
            sdz_approval_status: approval_status,
            sdz_park_attributes: park_attributes,
            sdz_street_attributes: street_attributes,
            sdz_instagram_tag: instagram_tag,
            sdz_instagram_location_url: instagram_location_url,
            sdz_instagram_profile_url: instagram_profile_url,
            sdz_google_place_id: google_place_id,
            sdz_google_maps_url: google_maps_url,
            sdz_address: address,
            sdz_phone_number: phone_number,
            sdz_google_rating: google_rating,
            sdz_google_rating_count: google_rating_count,
            sdz_google_types: google_types,
            sdz_user_id: self.sdz_user_id.clone(),
            created_at: self.created_at,
            updated_at: sdz_now_jst(),
        })
    }
}

pub fn sdz_validate_spot(
    name: &str,
    location: Option<&SdzSpotLocation>,
    tags: &[String],
    images: &[String],
    park_attributes: Option<&SdzSpotParkAttributes>,
    street_attributes: Option<&SdzStreetAttributes>,
) -> Result<(), SdzSpotValidationError> {
    if name.trim().is_empty() {
        return Err(SdzSpotValidationError::NameIsRequired);
    }
    if let Some(loc) = location {
        if !(loc.lat >= -90.0 && loc.lat <= 90.0) {
            return Err(SdzSpotValidationError::InvalidLatitude);
        }
        if !(loc.lng >= -180.0 && loc.lng <= 180.0) {
            return Err(SdzSpotValidationError::InvalidLongitude);
        }
    }
    if tags.len() > 10 {
        return Err(SdzSpotValidationError::TooManyTags);
    }
    if images.len() > SDZ_MAX_IMAGES_PER_SPOT {
        return Err(SdzSpotValidationError::TooManyImages);
    }
    if let Some(attrs) = park_attributes {
        if let Some(hours) = &attrs.business_hours {
            validate_business_hours(hours)?;
        }
    }
    if let Some(attrs) = street_attributes {
        if let Some(sections) = &attrs.sections {
            for section in sections {
                if section.section_type.trim().is_empty() {
                    return Err(SdzSpotValidationError::InvalidStreetSection);
                }
            }
        }
    }
    Ok(())
}

fn sdz_validate_google_rating(rating: f64) -> Result<(), SdzSpotValidationError> {
    if !(1.0..=5.0).contains(&rating) {
        return Err(SdzSpotValidationError::InvalidGoogleRating);
    }
    Ok(())
}

pub fn sdz_now_jst() -> DateTime<FixedOffset> {
    let offset = FixedOffset::east_opt(9 * 3600).expect("valid offset");
    chrono::Utc::now().with_timezone(&offset)
}

#[derive(Debug, Error)]
pub enum SdzSpotValidationError {
    #[error("name is required")]
    NameIsRequired,
    #[error("lat must be between -90 and 90")]
    InvalidLatitude,
    #[error("lng must be between -180 and 180")]
    InvalidLongitude,
    #[error("tags must be <= 10 items")]
    TooManyTags,
    #[error("images must be <= 3 items")]
    TooManyImages,
    #[error("business hours are invalid")]
    InvalidBusinessHours,
    #[error("street section type is required")]
    InvalidStreetSection,
    #[error("google rating must be between 1.0 and 5.0")]
    InvalidGoogleRating,
}

const SDZ_MAX_IMAGES_PER_SPOT: usize = 3;

fn validate_business_hours(hours: &SdzSpotBusinessHours) -> Result<(), SdzSpotValidationError> {
    let schedule_type = hours
        .schedule_type
        .clone()
        .unwrap_or(SdzSpotBusinessScheduleType::Regular);

    match schedule_type {
        SdzSpotBusinessScheduleType::Regular => {
            if hours.is_24_hours {
                return Ok(());
            }
            if let Some(weekday) = hours.weekday.as_ref() {
                validate_time_range(weekday)?;
            } else {
                return Err(SdzSpotValidationError::InvalidBusinessHours);
            }
            if !hours.same_as_weekday {
                if let Some(weekend) = hours.weekend.as_ref() {
                    validate_time_range(weekend)?;
                } else {
                    return Err(SdzSpotValidationError::InvalidBusinessHours);
                }
            }
            Ok(())
        }
        SdzSpotBusinessScheduleType::WeekdayOnly => {
            if hours.is_24_hours {
                return Ok(());
            }
            if let Some(weekday) = hours.weekday.as_ref() {
                validate_time_range(weekday)?;
                Ok(())
            } else {
                Err(SdzSpotValidationError::InvalidBusinessHours)
            }
        }
        SdzSpotBusinessScheduleType::WeekendOnly => {
            if hours.is_24_hours {
                return Ok(());
            }
            if let Some(weekend) = hours.weekend.as_ref() {
                validate_time_range(weekend)?;
                Ok(())
            } else {
                Err(SdzSpotValidationError::InvalidBusinessHours)
            }
        }
        SdzSpotBusinessScheduleType::Irregular
        | SdzSpotBusinessScheduleType::SchoolOnly
        | SdzSpotBusinessScheduleType::Manual => {
            if hours
                .note
                .as_ref()
                .map(|note| !note.trim().is_empty())
                .unwrap_or(false)
            {
                Ok(())
            } else {
                Err(SdzSpotValidationError::InvalidBusinessHours)
            }
        }
    }
}

fn validate_time_range(range: &SdzSpotTimeRange) -> Result<(), SdzSpotValidationError> {
    if range.start_minutes >= range.end_minutes {
        return Err(SdzSpotValidationError::InvalidBusinessHours);
    }
    if range.end_minutes > 24 * 60 {
        return Err(SdzSpotValidationError::InvalidBusinessHours);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn build_create_params() -> SdzCreateSpotParams {
        SdzCreateSpotParams {
            sdz_spot_id: "test-1".into(),
            name: "test spot".into(),
            description: Some("desc".into()),
            location: Some(SdzSpotLocation {
                lat: 35.0,
                lng: 139.0,
            }),
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
        }
    }

    #[test]
    fn create_spot_with_params() {
        let spot = SdzSpot::new_with_id(build_create_params()).unwrap();
        assert_eq!(spot.name, "test spot");
        assert_eq!(spot.sdz_user_id, "user-1");
        assert!(spot.sdz_google_place_id.is_none());
        assert!(spot.sdz_google_types.is_empty());
    }

    #[test]
    fn create_spot_with_google_places() {
        let mut params = build_create_params();
        params.sdz_google_place_id = Some("ChIJ_example".into());
        params.sdz_google_rating = Some(4.5);
        params.sdz_google_rating_count = Some(120);
        params.sdz_google_types = vec!["park".into(), "point_of_interest".into()];

        let spot = SdzSpot::new_with_id(params).unwrap();
        assert_eq!(spot.sdz_google_place_id, Some("ChIJ_example".into()));
        assert_eq!(spot.sdz_google_rating, Some(4.5));
        assert_eq!(spot.sdz_google_rating_count, Some(120));
        assert_eq!(spot.sdz_google_types.len(), 2);
    }

    #[test]
    fn reject_invalid_google_rating() {
        let mut params = build_create_params();
        params.sdz_google_rating = Some(5.5);
        let err = SdzSpot::new_with_id(params).unwrap_err();
        assert!(err.to_string().contains("google rating"));
    }

    #[test]
    fn update_spot_with_params() {
        let spot = SdzSpot::new_with_id(build_create_params()).unwrap();
        let updated = spot
            .update(SdzUpdateSpotParams {
                name: Some("updated".into()),
                sdz_google_place_id: Some("ChIJ_new".into()),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(updated.name, "updated");
        assert_eq!(updated.sdz_google_place_id, Some("ChIJ_new".into()));
        assert_eq!(updated.description, Some("desc".into()));
    }
}
