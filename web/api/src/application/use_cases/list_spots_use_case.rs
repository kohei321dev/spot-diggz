use std::sync::Arc;

use crate::{
    application::use_cases::spot_repository::SdzSpotRepository, domain::models::SdzSpot,
    presentation::error::SdzApiError,
};

const SDZ_PARK_TAGS: [&str; 3] = ["パーク", "スケートパーク", "スケートボードパーク"];
const SDZ_STREET_TAGS: [&str; 1] = ["ストリート"];

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SdzSpotTypeFilter {
    Park,
    Street,
}

impl SdzSpotTypeFilter {
    pub fn parse(raw: &str) -> Option<Self> {
        match raw {
            "park" | "skatepark" => Some(Self::Park),
            "street" => Some(Self::Street),
            _ => None,
        }
    }
}

#[derive(Debug, Clone, Default)]
pub struct SdzSpotSearchFilter {
    pub query: Option<String>,
    pub spot_type: Option<SdzSpotTypeFilter>,
    pub tags: Vec<String>,
}

impl SdzSpotSearchFilter {
    pub fn is_active(&self) -> bool {
        self.query
            .as_ref()
            .map(|q| !q.trim().is_empty())
            .unwrap_or(false)
            || self.spot_type.is_some()
            || !self.tags.is_empty()
    }

    pub fn apply(&self, spots: Vec<SdzSpot>) -> Vec<SdzSpot> {
        let normalized_query = self
            .query
            .as_ref()
            .map(|q| q.trim())
            .filter(|q| !q.is_empty())
            .map(|q| q.to_lowercase());

        spots
            .into_iter()
            .filter(|spot| {
                matches_query(spot, normalized_query.as_deref())
                    && matches_type(spot, self.spot_type)
                    && matches_tags(spot, &self.tags)
            })
            .collect()
    }
}

pub struct SdzListSpotsUseCase;

impl SdzListSpotsUseCase {
    pub fn new() -> Self {
        Self
    }

    pub async fn execute(
        &self,
        repo: Arc<dyn SdzSpotRepository>,
        limit: usize,
        viewer_user_id: Option<String>,
        filter: SdzSpotSearchFilter,
    ) -> Result<Vec<SdzSpot>, SdzApiError> {
        let capped = limit.min(100); // 念のため上限
        let fetch_limit = if filter.is_active() {
            (capped.saturating_mul(4)).min(200)
        } else {
            capped
        };
        let mut spots = repo.list_recent(fetch_limit).await?;
        if let Some(user_id) = viewer_user_id {
            spots.retain(|spot| spot.is_approved() || spot.sdz_user_id == user_id);
        } else {
            spots.retain(|spot| spot.is_approved());
        }
        let mut spots = filter.apply(spots);
        spots.truncate(capped);
        Ok(spots)
    }
}

fn matches_query(spot: &SdzSpot, normalized_query: Option<&str>) -> bool {
    let Some(query) = normalized_query else {
        return true;
    };
    let name_match = spot.name.to_lowercase().contains(query);
    let desc_match = spot
        .description
        .as_ref()
        .map(|desc| desc.to_lowercase().contains(query))
        .unwrap_or(false);
    let tag_match = spot
        .tags
        .iter()
        .any(|tag| tag.to_lowercase().contains(query));

    name_match || desc_match || tag_match
}

fn matches_type(spot: &SdzSpot, spot_type: Option<SdzSpotTypeFilter>) -> bool {
    let Some(spot_type) = spot_type else {
        return true;
    };

    match spot_type {
        SdzSpotTypeFilter::Park => {
            spot.sdz_park_attributes.is_some() || has_any_tag(spot, &SDZ_PARK_TAGS)
        }
        SdzSpotTypeFilter::Street => {
            spot.sdz_street_attributes.is_some() || has_any_tag(spot, &SDZ_STREET_TAGS)
        }
    }
}

fn matches_tags(spot: &SdzSpot, tags: &[String]) -> bool {
    if tags.is_empty() {
        return true;
    }
    tags.iter()
        .any(|tag| spot.tags.iter().any(|spot_tag| spot_tag == tag))
}

fn has_any_tag(spot: &SdzSpot, tags: &[&str]) -> bool {
    spot.tags.iter().any(|tag| tags.contains(&tag.as_str()))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        domain::models::{SdzSpotApprovalStatus, SdzSpotParkAttributes, SdzStreetAttributes},
        infrastructure::in_memory_spot_repository::SdzInMemorySpotRepository,
    };
    use chrono::{FixedOffset, TimeZone};

    #[tokio::test]
    async fn list_spots_returns_empty() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let use_case = SdzListSpotsUseCase::new();
        let list = use_case
            .execute(repo, 10, None, SdzSpotSearchFilter::default())
            .await
            .unwrap();
        assert!(list.is_empty());
    }

    #[tokio::test]
    async fn list_spots_orders_recent() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let tz = FixedOffset::east_opt(9 * 3600).unwrap();
        // 1件だけ入れて確認
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-1".into(),
            name: "A".into(),
            description: None,
            location: None,
            tags: vec![],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();

        let use_case = SdzListSpotsUseCase::new();
        let list = use_case
            .execute(repo, 10, None, SdzSpotSearchFilter::default())
            .await
            .unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].sdz_spot_id, "spot-1");
    }

    #[tokio::test]
    async fn list_spots_includes_owner_unapproved() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let tz = FixedOffset::east_opt(9 * 3600).unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-owner".into(),
            name: "Owner".into(),
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
            created_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-other".into(),
            name: "Other".into(),
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
            sdz_user_id: "user-2".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 3, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 3, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();

        let use_case = SdzListSpotsUseCase::new();
        let list = use_case
            .execute(
                repo,
                10,
                Some("user-1".into()),
                SdzSpotSearchFilter::default(),
            )
            .await
            .unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].sdz_spot_id, "spot-owner");
    }

    #[tokio::test]
    async fn list_spots_filters_by_query() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let tz = FixedOffset::east_opt(9 * 3600).unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-1".into(),
            name: "Osaka Park".into(),
            description: Some("Nice ledge".into()),
            location: None,
            tags: vec!["パーク".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-2".into(),
            name: "Tokyo Street".into(),
            description: None,
            location: None,
            tags: vec!["ストリート".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();

        let use_case = SdzListSpotsUseCase::new();
        let list = use_case
            .execute(
                repo,
                10,
                None,
                SdzSpotSearchFilter {
                    query: Some("ledge".into()),
                    spot_type: None,
                    tags: vec![],
                },
            )
            .await
            .unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].sdz_spot_id, "spot-1");
    }

    #[tokio::test]
    async fn list_spots_filters_by_type_and_tags() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let tz = FixedOffset::east_opt(9 * 3600).unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-park".into(),
            name: "Park".into(),
            description: None,
            location: None,
            tags: vec!["パーク".into(), "夜".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
            sdz_park_attributes: Some(SdzSpotParkAttributes {
                official_url: None,
                business_hours: None,
                access_info: None,
                phone_number: None,
            }),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-street".into(),
            name: "Street".into(),
            description: None,
            location: None,
            tags: vec!["ストリート".into(), "夜".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
            sdz_park_attributes: None,
            sdz_street_attributes: Some(SdzStreetAttributes {
                surface_material: None,
                surface_condition: None,
                sections: None,
                difficulty: None,
                notes: None,
            }),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();

        let use_case = SdzListSpotsUseCase::new();
        let list = use_case
            .execute(
                repo,
                10,
                None,
                SdzSpotSearchFilter {
                    query: None,
                    spot_type: Some(SdzSpotTypeFilter::Street),
                    tags: vec!["夜".into()],
                },
            )
            .await
            .unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].sdz_spot_id, "spot-street");
    }

    #[tokio::test]
    async fn list_spots_filters_by_tags_or() {
        let repo = Arc::new(SdzInMemorySpotRepository::default());
        let tz = FixedOffset::east_opt(9 * 3600).unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-night".into(),
            name: "Night".into(),
            description: None,
            location: None,
            tags: vec!["夜".into(), "パーク".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 1, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-morning".into(),
            name: "Morning".into(),
            description: None,
            location: None,
            tags: vec!["朝".into(), "ストリート".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 2, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();
        repo.create(crate::domain::models::SdzSpot {
            sdz_spot_id: "spot-none".into(),
            name: "None".into(),
            description: None,
            location: None,
            tags: vec!["雨".into()],
            images: vec![],
            sdz_approval_status: Some(SdzSpotApprovalStatus::Approved),
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
            sdz_user_id: "user".into(),
            created_at: tz.with_ymd_and_hms(2024, 1, 3, 0, 0, 0).unwrap(),
            updated_at: tz.with_ymd_and_hms(2024, 1, 3, 0, 0, 0).unwrap(),
        })
        .await
        .unwrap();

        let use_case = SdzListSpotsUseCase::new();
        let list = use_case
            .execute(
                repo,
                10,
                None,
                SdzSpotSearchFilter {
                    query: None,
                    spot_type: None,
                    tags: vec!["夜".into(), "朝".into()],
                },
            )
            .await
            .unwrap();
        let ids = list
            .iter()
            .map(|spot| spot.sdz_spot_id.as_str())
            .collect::<Vec<_>>();
        assert_eq!(list.len(), 2);
        assert!(ids.contains(&"spot-night"));
        assert!(ids.contains(&"spot-morning"));
    }
}
