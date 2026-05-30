use async_trait::async_trait;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::{json, Map};

use crate::{
    application::use_cases::spot_repository::SdzSpotRepository,
    domain::models::{
        SdzSpot, SdzSpotApprovalStatus, SdzSpotBusinessHours, SdzSpotBusinessScheduleType,
        SdzSpotLocation, SdzSpotParkAttributes, SdzSpotTimeRange, SdzStreetAttributes,
        SdzStreetSection, SdzStreetSurfaceCondition,
    },
    presentation::error::SdzApiError,
};

pub struct SdzFirestoreSpotRepository {
    project_id: String,
    bearer_token: Option<String>,
    http: Client,
}

impl SdzFirestoreSpotRepository {
    pub fn new(project_id: String, bearer_token: Option<String>) -> Result<Self, SdzApiError> {
        let http = Client::builder().build().map_err(|e| {
            tracing::error!("Failed to build reqwest client: {:?}", e);
            SdzApiError::Internal
        })?;
        Ok(Self {
            project_id,
            bearer_token,
            http,
        })
    }

    fn resolve_token(&self) -> &str {
        self.bearer_token.as_deref().unwrap_or("")
    }

    async fn upsert_document(&self, spot: &SdzSpot) -> Result<(), SdzApiError> {
        let url = format!(
            "https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents/spots/{}",
            self.project_id, spot.sdz_spot_id
        );

        let body = build_firestore_doc(spot)?;

        let resp = self
            .http
            .patch(url)
            .bearer_auth(self.resolve_token())
            .json(&body)
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Firestore request error: {:?}", e);
                SdzApiError::Internal
            })?;

        map_status(resp.status(), resp.text().await).map(|_| ())
    }

    async fn get_document(&self, spot_id: &str) -> Result<Option<FirestoreSpotDoc>, SdzApiError> {
        let url = format!(
            "https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents/spots/{}",
            self.project_id, spot_id
        );
        let resp = self
            .http
            .get(url)
            .bearer_auth(self.resolve_token())
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Firestore request error: {:?}", e);
                SdzApiError::Internal
            })?;

        match resp.status() {
            reqwest::StatusCode::OK => {
                let doc = resp.json::<FirestoreSpotDoc>().await.map_err(|e| {
                    tracing::error!("Failed to parse Firestore response: {:?}", e);
                    SdzApiError::Internal
                })?;
                Ok(Some(doc))
            }
            reqwest::StatusCode::NOT_FOUND => Ok(None),
            code => {
                let body = resp.text().await.unwrap_or_default();
                tracing::error!("Firestore unexpected status: {} body: {}", code, body);
                Err(SdzApiError::Internal)
            }
        }
    }
}

#[async_trait]
impl SdzSpotRepository for SdzFirestoreSpotRepository {
    async fn create(&self, spot: SdzSpot) -> Result<SdzSpot, SdzApiError> {
        self.upsert_document(&spot).await?;
        Ok(spot)
    }

    async fn update(&self, spot: SdzSpot) -> Result<SdzSpot, SdzApiError> {
        self.upsert_document(&spot).await?;
        Ok(spot)
    }

    async fn find_by_id(&self, spot_id: &str) -> Result<Option<SdzSpot>, SdzApiError> {
        let Some(doc) = self.get_document(spot_id).await? else {
            return Ok(None);
        };

        Ok(Some(doc.into_spot(spot_id.to_string())))
    }

    async fn list_recent(&self, _limit: usize) -> Result<Vec<SdzSpot>, SdzApiError> {
        let limit = _limit.min(100) as i32;
        let url = format!(
            "https://firestore.googleapis.com/v1/projects/{}/databases/(default)/documents:runQuery",
            self.project_id
        );

        let body = json!({
            "structuredQuery": {
                "from": [{ "collectionId": "spots" }],
                "orderBy": [{
                    "field": { "fieldPath": "createdAt" },
                    "direction": "DESCENDING"
                }],
                "limit": limit
            }
        });

        let resp = self
            .http
            .post(url)
            .bearer_auth(self.resolve_token())
            .json(&body)
            .send()
            .await
            .map_err(|e| {
                tracing::error!("Firestore runQuery request error: {:?}", e);
                SdzApiError::Internal
            })?;

        if !resp.status().is_success() {
            let status = resp.status();
            let text = resp.text().await.unwrap_or_default();
            tracing::error!(
                "Firestore runQuery unexpected status {} body: {}",
                status,
                text
            );
            return Err(SdzApiError::Internal);
        }

        let rows = resp
            .json::<Vec<FirestoreRunQueryResponse>>()
            .await
            .map_err(|e| {
                tracing::error!("Failed to parse Firestore runQuery response: {:?}", e);
                SdzApiError::Internal
            })?;

        let mut spots = Vec::new();
        for row in rows {
            if let Some(doc) = row.document {
                if let Some(spot_id) = extract_doc_id(&doc.name) {
                    spots.push(doc.into_spot(spot_id));
                }
            }
        }
        Ok(spots)
    }

    async fn count_image_spots_by_user(&self, _user_id: &str) -> Result<usize, SdzApiError> {
        // Firestore側では簡易実装: 全取得してフィルタ
        // 今後必要に応じてFirestoreクエリで最適化
        Ok(0)
    }
}

// ─── 書き込み: SdzSpot → Firestore ───

fn build_firestore_doc(spot: &SdzSpot) -> Result<serde_json::Value, SdzApiError> {
    let mut fields = Map::new();

    fields.insert("name".into(), string_value(&spot.name));
    if let Some(desc) = &spot.description {
        fields.insert("description".into(), string_value(desc));
    }
    fields.insert("userId".into(), string_value(&spot.sdz_user_id));

    insert_string_array(&mut fields, "tags", &spot.tags);
    insert_string_array(&mut fields, "images", &spot.images);

    if let Some(status) = &spot.sdz_approval_status {
        fields.insert(
            "approvalStatus".into(),
            string_value(approval_status_as_str(status)),
        );
    }

    if let Some(attrs) = &spot.sdz_park_attributes {
        if let Some(value) = build_park_attributes(attrs) {
            fields.insert("parkAttributes".into(), value);
        }
    }

    if let Some(attrs) = &spot.sdz_street_attributes {
        if let Some(value) = build_street_attributes(attrs) {
            fields.insert("streetAttributes".into(), value);
        }
    }

    if let Some(tag) = &spot.sdz_instagram_tag {
        fields.insert("instagramTag".into(), string_value(tag));
    }
    if let Some(url) = &spot.sdz_instagram_location_url {
        fields.insert("instagramLocationUrl".into(), string_value(url));
    }
    if let Some(url) = &spot.sdz_instagram_profile_url {
        fields.insert("instagramProfileUrl".into(), string_value(url));
    }

    if let Some(place_id) = &spot.sdz_google_place_id {
        fields.insert("googlePlaceId".into(), string_value(place_id));
    }
    if let Some(url) = &spot.sdz_google_maps_url {
        fields.insert("googleMapsUrl".into(), string_value(url));
    }
    if let Some(addr) = &spot.sdz_address {
        fields.insert("address".into(), string_value(addr));
    }
    if let Some(phone) = &spot.sdz_phone_number {
        fields.insert("phoneNumber".into(), string_value(phone));
    }
    if let Some(rating) = spot.sdz_google_rating {
        fields.insert("googleRating".into(), double_value(rating));
    }
    if let Some(count) = spot.sdz_google_rating_count {
        fields.insert("googleRatingCount".into(), integer_value(count));
    }
    insert_string_array(&mut fields, "googleTypes", &spot.sdz_google_types);

    if let Some(loc) = &spot.location {
        fields.insert(
            "location".into(),
            json!({
                "mapValue": {
                    "fields": {
                        "lat": { "doubleValue": loc.lat },
                        "lng": { "doubleValue": loc.lng }
                    }
                }
            }),
        );
    }

    fields.insert(
        "createdAt".into(),
        json!({ "timestampValue": spot.created_at.to_rfc3339() }),
    );
    fields.insert(
        "updatedAt".into(),
        json!({ "timestampValue": spot.updated_at.to_rfc3339() }),
    );

    Ok(json!({ "fields": fields }))
}

fn build_park_attributes(attrs: &SdzSpotParkAttributes) -> Option<serde_json::Value> {
    let mut fields = Map::new();
    if let Some(url) = &attrs.official_url {
        fields.insert("officialUrl".into(), string_value(url));
    }
    if let Some(hours) = &attrs.business_hours {
        fields.insert("businessHours".into(), build_business_hours(hours));
    }
    if let Some(info) = &attrs.access_info {
        fields.insert("accessInfo".into(), string_value(info));
    }
    if let Some(phone) = &attrs.phone_number {
        fields.insert("phoneNumber".into(), string_value(phone));
    }
    if fields.is_empty() {
        return None;
    }
    Some(map_value(fields))
}

fn build_business_hours(hours: &SdzSpotBusinessHours) -> serde_json::Value {
    let mut fields = Map::new();
    if let Some(schedule_type) = &hours.schedule_type {
        fields.insert(
            "scheduleType".into(),
            string_value(schedule_type_as_str(schedule_type)),
        );
    }
    fields.insert("is24Hours".into(), bool_value(hours.is_24_hours));
    fields.insert("sameAsWeekday".into(), bool_value(hours.same_as_weekday));
    if let Some(weekday) = &hours.weekday {
        fields.insert("weekday".into(), build_time_range(weekday));
    }
    if let Some(weekend) = &hours.weekend {
        fields.insert("weekend".into(), build_time_range(weekend));
    }
    if let Some(note) = &hours.note {
        fields.insert("note".into(), string_value(note));
    }
    map_value(fields)
}

fn build_time_range(range: &SdzSpotTimeRange) -> serde_json::Value {
    let mut fields = Map::new();
    fields.insert("startMinutes".into(), integer_value(range.start_minutes));
    fields.insert("endMinutes".into(), integer_value(range.end_minutes));
    map_value(fields)
}

fn build_street_attributes(attrs: &SdzStreetAttributes) -> Option<serde_json::Value> {
    let mut fields = Map::new();
    if let Some(material) = &attrs.surface_material {
        fields.insert("surfaceMaterial".into(), string_value(material));
    }
    if let Some(condition) = &attrs.surface_condition {
        let mut cond_fields = Map::new();
        if let Some(roughness) = &condition.roughness {
            cond_fields.insert("roughness".into(), string_value(roughness));
        }
        if let Some(crack) = &condition.crack {
            cond_fields.insert("crack".into(), string_value(crack));
        }
        if !cond_fields.is_empty() {
            fields.insert("surfaceCondition".into(), map_value(cond_fields));
        }
    }
    if let Some(sections) = &attrs.sections {
        let values: Vec<serde_json::Value> = sections.iter().filter_map(build_section).collect();
        if !values.is_empty() {
            fields.insert(
                "sections".into(),
                json!({ "arrayValue": { "values": values } }),
            );
        }
    }
    if let Some(difficulty) = &attrs.difficulty {
        fields.insert("difficulty".into(), string_value(difficulty));
    }
    if let Some(notes) = &attrs.notes {
        fields.insert("notes".into(), string_value(notes));
    }
    if fields.is_empty() {
        return None;
    }
    Some(map_value(fields))
}

fn build_section(section: &SdzStreetSection) -> Option<serde_json::Value> {
    if section.section_type.trim().is_empty() {
        return None;
    }
    let mut fields = Map::new();
    fields.insert("type".into(), string_value(&section.section_type));
    if let Some(count) = section.count {
        fields.insert("count".into(), integer_value(count));
    }
    if let Some(height) = section.height_cm {
        fields.insert("heightCm".into(), integer_value(height));
    }
    if let Some(width) = section.width_cm {
        fields.insert("widthCm".into(), integer_value(width));
    }
    if let Some(notes) = &section.notes {
        fields.insert("notes".into(), string_value(notes));
    }
    Some(map_value(fields))
}

fn string_value(s: &str) -> serde_json::Value {
    json!({ "stringValue": s })
}

fn double_value(v: f64) -> serde_json::Value {
    json!({ "doubleValue": v })
}

fn bool_value(v: bool) -> serde_json::Value {
    json!({ "booleanValue": v })
}

fn integer_value<T: std::fmt::Display>(v: T) -> serde_json::Value {
    json!({ "integerValue": v.to_string() })
}

fn map_value(fields: Map<String, serde_json::Value>) -> serde_json::Value {
    json!({ "mapValue": { "fields": fields } })
}

fn insert_string_array(fields: &mut Map<String, serde_json::Value>, key: &str, values: &[String]) {
    if !values.is_empty() {
        let arr: Vec<serde_json::Value> = values.iter().map(|s| string_value(s)).collect();
        fields.insert(key.into(), json!({ "arrayValue": { "values": arr } }));
    }
}

fn approval_status_as_str(status: &SdzSpotApprovalStatus) -> &'static str {
    match status {
        SdzSpotApprovalStatus::Pending => "pending",
        SdzSpotApprovalStatus::Approved => "approved",
        SdzSpotApprovalStatus::Rejected => "rejected",
    }
}

fn schedule_type_as_str(st: &SdzSpotBusinessScheduleType) -> &'static str {
    match st {
        SdzSpotBusinessScheduleType::Regular => "regular",
        SdzSpotBusinessScheduleType::WeekdayOnly => "weekdayOnly",
        SdzSpotBusinessScheduleType::WeekendOnly => "weekendOnly",
        SdzSpotBusinessScheduleType::Irregular => "irregular",
        SdzSpotBusinessScheduleType::SchoolOnly => "schoolOnly",
        SdzSpotBusinessScheduleType::Manual => "manual",
    }
}

// ─── 読み取り: Firestore → SdzSpot ───

#[derive(Debug, Serialize, Deserialize)]
struct FirestoreSpotDoc {
    fields: FirestoreSpotFields,
}

#[derive(Debug, Serialize, Deserialize)]
struct FirestoreSpotFields {
    name: Option<StringField>,
    description: Option<StringField>,
    #[serde(rename = "userId")]
    user_id: Option<StringField>,
    tags: Option<ArrayField>,
    images: Option<ArrayField>,
    #[serde(rename = "approvalStatus")]
    approval_status: Option<StringField>,
    #[serde(rename = "parkAttributes")]
    park_attributes: Option<GenericMapField>,
    #[serde(rename = "streetAttributes")]
    street_attributes: Option<GenericMapField>,
    #[serde(rename = "instagramTag")]
    instagram_tag: Option<StringField>,
    #[serde(rename = "instagramLocationUrl")]
    instagram_location_url: Option<StringField>,
    #[serde(rename = "instagramProfileUrl")]
    instagram_profile_url: Option<StringField>,
    #[serde(rename = "googlePlaceId")]
    google_place_id: Option<StringField>,
    #[serde(rename = "googleMapsUrl")]
    google_maps_url: Option<StringField>,
    address: Option<StringField>,
    #[serde(rename = "phoneNumber")]
    phone_number: Option<StringField>,
    #[serde(rename = "googleRating")]
    google_rating: Option<DoubleField>,
    #[serde(rename = "googleRatingCount")]
    google_rating_count: Option<IntegerField>,
    #[serde(rename = "googleTypes")]
    google_types: Option<ArrayField>,
    location: Option<GenericMapField>,
    #[serde(rename = "createdAt")]
    created_at: Option<TimestampField>,
    #[serde(rename = "updatedAt")]
    updated_at: Option<TimestampField>,
    // 旧スキーマ（後方互換読み取り用）
    #[serde(rename = "trustLevel")]
    trust_level: Option<StringField>,
    #[serde(rename = "instagramUrl")]
    instagram_url: Option<StringField>,
    #[serde(rename = "officialUrl")]
    official_url_legacy: Option<StringField>,
    #[serde(rename = "businessHours")]
    business_hours_legacy: Option<StringField>,
    #[serde(rename = "sections")]
    sections_legacy: Option<ArrayField>,
}

#[derive(Debug, Serialize, Deserialize)]
struct StringField {
    #[serde(rename = "stringValue")]
    string_value: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct DoubleField {
    #[serde(rename = "doubleValue")]
    double_value: f64,
}

#[derive(Debug, Serialize, Deserialize)]
struct IntegerField {
    #[serde(rename = "integerValue")]
    integer_value: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct ArrayField {
    #[serde(rename = "arrayValue")]
    array_value: ArrayValue,
}

#[derive(Debug, Serialize, Deserialize)]
struct ArrayValue {
    values: Option<Vec<serde_json::Value>>,
}

#[derive(Debug, Serialize, Deserialize)]
struct GenericMapField {
    #[serde(rename = "mapValue")]
    map_value: GenericMapValue,
}

#[derive(Debug, Serialize, Deserialize)]
struct GenericMapValue {
    fields: Option<serde_json::Map<String, serde_json::Value>>,
}

#[derive(Debug, Serialize, Deserialize)]
struct TimestampField {
    #[serde(rename = "timestampValue")]
    timestamp_value: String,
}

impl FirestoreSpotDoc {
    fn into_spot(self, spot_id: String) -> SdzSpot {
        let fields = self.fields;

        let tags = extract_string_array(fields.tags);
        let images = extract_string_array(fields.images);

        let location = fields.location.and_then(|loc| {
            let map_fields = loc.map_value.fields?;
            let lat = extract_double_from_map(&map_fields, "lat")?;
            let lng = extract_double_from_map(&map_fields, "lng")?;
            Some(SdzSpotLocation { lat, lng })
        });

        // approvalStatus: 新スキーマ優先 → 旧 trustLevel からフォールバック
        let approval_status = fields
            .approval_status
            .map(|s| s.string_value)
            .or_else(|| {
                fields.trust_level.map(|tl| match tl.string_value.as_str() {
                    "verified" => "approved".to_string(),
                    _ => "pending".to_string(),
                })
            })
            .and_then(|s| parse_approval_status(&s));

        // parkAttributes: 新スキーマ優先 → 旧フラットフィールドからフォールバック
        let park_attributes = fields
            .park_attributes
            .and_then(|m| parse_park_attributes_map(m.map_value.fields))
            .or_else(|| {
                let official_url = fields.official_url_legacy.map(|s| s.string_value);
                let bh_note = fields.business_hours_legacy.map(|s| s.string_value);
                if official_url.is_some() || bh_note.is_some() {
                    Some(SdzSpotParkAttributes {
                        official_url,
                        business_hours: bh_note.map(|note| SdzSpotBusinessHours {
                            schedule_type: Some(SdzSpotBusinessScheduleType::Manual),
                            is_24_hours: false,
                            same_as_weekday: false,
                            weekday: None,
                            weekend: None,
                            note: Some(note),
                        }),
                        access_info: None,
                        phone_number: None,
                    })
                } else {
                    None
                }
            });

        // streetAttributes: 新スキーマ優先 → 旧 sections からフォールバック
        let street_attributes = fields
            .street_attributes
            .and_then(|m| parse_street_attributes_map(m.map_value.fields))
            .or_else(|| {
                let old_sections = extract_string_array(fields.sections_legacy);
                if old_sections.is_empty() {
                    None
                } else {
                    Some(SdzStreetAttributes {
                        surface_material: None,
                        surface_condition: None,
                        sections: Some(
                            old_sections
                                .into_iter()
                                .map(|s| SdzStreetSection {
                                    section_type: s,
                                    count: None,
                                    height_cm: None,
                                    width_cm: None,
                                    notes: None,
                                })
                                .collect(),
                        ),
                        difficulty: None,
                        notes: None,
                    })
                }
            });

        // Instagram: 新スキーマ優先 → 旧 instagramUrl からフォールバック
        let instagram_tag = fields
            .instagram_tag
            .map(|s| s.string_value)
            .or_else(|| fields.instagram_url.map(|s| s.string_value));

        let google_types = fields
            .google_types
            .map(|f| extract_string_array(Some(f)))
            .unwrap_or_default();

        let created_at = parse_timestamp(fields.created_at.map(|t| t.timestamp_value))
            .unwrap_or_else(|| chrono::Utc::now().into());
        let updated_at = parse_timestamp(fields.updated_at.map(|t| t.timestamp_value))
            .unwrap_or_else(|| chrono::Utc::now().into());

        SdzSpot {
            sdz_spot_id: spot_id,
            name: fields
                .name
                .map(|s| s.string_value)
                .unwrap_or_else(|| "unknown".into()),
            description: fields.description.map(|s| s.string_value),
            location,
            tags,
            images,
            sdz_approval_status: approval_status,
            sdz_park_attributes: park_attributes,
            sdz_street_attributes: street_attributes,
            sdz_instagram_tag: instagram_tag,
            sdz_instagram_location_url: fields.instagram_location_url.map(|s| s.string_value),
            sdz_instagram_profile_url: fields.instagram_profile_url.map(|s| s.string_value),
            sdz_google_place_id: fields.google_place_id.map(|s| s.string_value),
            sdz_google_maps_url: fields.google_maps_url.map(|s| s.string_value),
            sdz_address: fields.address.map(|s| s.string_value),
            sdz_phone_number: fields.phone_number.map(|s| s.string_value),
            sdz_google_rating: fields.google_rating.map(|d| d.double_value),
            sdz_google_rating_count: fields
                .google_rating_count
                .and_then(|i| i.integer_value.parse::<u32>().ok()),
            sdz_google_types: google_types,
            sdz_user_id: fields.user_id.map(|s| s.string_value).unwrap_or_default(),
            created_at,
            updated_at,
        }
    }
}

fn parse_park_attributes_map(
    fields: Option<serde_json::Map<String, serde_json::Value>>,
) -> Option<SdzSpotParkAttributes> {
    let fields = fields?;
    let official_url = extract_string_from_map(&fields, "officialUrl");
    let business_hours = fields
        .get("businessHours")
        .and_then(|v| v.get("mapValue"))
        .and_then(|mv| mv.get("fields"))
        .and_then(|f| f.as_object())
        .and_then(parse_business_hours_map);
    let access_info = extract_string_from_map(&fields, "accessInfo");
    let phone_number = extract_string_from_map(&fields, "phoneNumber");

    if official_url.is_none()
        && business_hours.is_none()
        && access_info.is_none()
        && phone_number.is_none()
    {
        return None;
    }
    Some(SdzSpotParkAttributes {
        official_url,
        business_hours,
        access_info,
        phone_number,
    })
}

fn parse_business_hours_map(
    fields: &serde_json::Map<String, serde_json::Value>,
) -> Option<SdzSpotBusinessHours> {
    let schedule_type =
        extract_string_from_map(fields, "scheduleType").and_then(|s| parse_schedule_type(&s));
    let is_24_hours = extract_bool_from_map(fields, "is24Hours").unwrap_or(false);
    let same_as_weekday = extract_bool_from_map(fields, "sameAsWeekday").unwrap_or(false);
    let weekday = fields
        .get("weekday")
        .and_then(|v| v.get("mapValue"))
        .and_then(|mv| mv.get("fields"))
        .and_then(|f| f.as_object())
        .and_then(parse_time_range_map);
    let weekend = fields
        .get("weekend")
        .and_then(|v| v.get("mapValue"))
        .and_then(|mv| mv.get("fields"))
        .and_then(|f| f.as_object())
        .and_then(parse_time_range_map);
    let note = extract_string_from_map(fields, "note");

    Some(SdzSpotBusinessHours {
        schedule_type,
        is_24_hours,
        same_as_weekday,
        weekday,
        weekend,
        note,
    })
}

fn parse_time_range_map(
    fields: &serde_json::Map<String, serde_json::Value>,
) -> Option<SdzSpotTimeRange> {
    let start = extract_integer_from_map(fields, "startMinutes")?;
    let end = extract_integer_from_map(fields, "endMinutes")?;
    Some(SdzSpotTimeRange {
        start_minutes: start as u16,
        end_minutes: end as u16,
    })
}

fn parse_street_attributes_map(
    fields: Option<serde_json::Map<String, serde_json::Value>>,
) -> Option<SdzStreetAttributes> {
    let fields = fields?;
    let surface_material = extract_string_from_map(&fields, "surfaceMaterial");
    let surface_condition = fields
        .get("surfaceCondition")
        .and_then(|v| v.get("mapValue"))
        .and_then(|mv| mv.get("fields"))
        .and_then(|f| f.as_object())
        .map(|f| SdzStreetSurfaceCondition {
            roughness: extract_string_from_map(f, "roughness"),
            crack: extract_string_from_map(f, "crack"),
        });
    let sections = fields
        .get("sections")
        .and_then(|v| v.get("arrayValue"))
        .and_then(|av| av.get("values"))
        .and_then(|vals| vals.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(parse_section_value)
                .collect::<Vec<_>>()
        });
    let difficulty = extract_string_from_map(&fields, "difficulty");
    let notes = extract_string_from_map(&fields, "notes");

    if surface_material.is_none()
        && surface_condition.is_none()
        && sections.is_none()
        && difficulty.is_none()
        && notes.is_none()
    {
        return None;
    }
    Some(SdzStreetAttributes {
        surface_material,
        surface_condition,
        sections,
        difficulty,
        notes,
    })
}

fn parse_section_value(value: &serde_json::Value) -> Option<SdzStreetSection> {
    let fields = value
        .get("mapValue")
        .and_then(|mv| mv.get("fields"))
        .and_then(|f| f.as_object())?;
    let section_type = extract_string_from_map(fields, "type")?;
    Some(SdzStreetSection {
        section_type,
        count: extract_integer_from_map(fields, "count").map(|v| v as u16),
        height_cm: extract_integer_from_map(fields, "heightCm").map(|v| v as u16),
        width_cm: extract_integer_from_map(fields, "widthCm").map(|v| v as u16),
        notes: extract_string_from_map(fields, "notes"),
    })
}

fn extract_string_from_map(
    fields: &serde_json::Map<String, serde_json::Value>,
    key: &str,
) -> Option<String> {
    fields
        .get(key)
        .and_then(|v| v.get("stringValue"))
        .and_then(|v| v.as_str())
        .map(|s| s.to_string())
}

fn extract_double_from_map(
    fields: &serde_json::Map<String, serde_json::Value>,
    key: &str,
) -> Option<f64> {
    fields
        .get(key)
        .and_then(|v| v.get("doubleValue"))
        .and_then(|v| v.as_f64())
}

fn extract_bool_from_map(
    fields: &serde_json::Map<String, serde_json::Value>,
    key: &str,
) -> Option<bool> {
    fields
        .get(key)
        .and_then(|v| v.get("booleanValue"))
        .and_then(|v| v.as_bool())
}

fn extract_integer_from_map(
    fields: &serde_json::Map<String, serde_json::Value>,
    key: &str,
) -> Option<i64> {
    fields
        .get(key)
        .and_then(|v| v.get("integerValue"))
        .and_then(|v| v.as_str())
        .and_then(|s| s.parse::<i64>().ok())
}

fn extract_string_array(field: Option<ArrayField>) -> Vec<String> {
    field
        .and_then(|f| f.array_value.values)
        .unwrap_or_default()
        .into_iter()
        .filter_map(|v| {
            v.get("stringValue")
                .and_then(|s| s.as_str())
                .map(|s| s.to_string())
        })
        .collect()
}

fn parse_approval_status(s: &str) -> Option<SdzSpotApprovalStatus> {
    match s {
        "pending" => Some(SdzSpotApprovalStatus::Pending),
        "approved" => Some(SdzSpotApprovalStatus::Approved),
        "rejected" => Some(SdzSpotApprovalStatus::Rejected),
        _ => None,
    }
}

fn parse_schedule_type(s: &str) -> Option<SdzSpotBusinessScheduleType> {
    match s {
        "regular" => Some(SdzSpotBusinessScheduleType::Regular),
        "weekdayOnly" => Some(SdzSpotBusinessScheduleType::WeekdayOnly),
        "weekendOnly" => Some(SdzSpotBusinessScheduleType::WeekendOnly),
        "irregular" => Some(SdzSpotBusinessScheduleType::Irregular),
        "schoolOnly" => Some(SdzSpotBusinessScheduleType::SchoolOnly),
        "manual" => Some(SdzSpotBusinessScheduleType::Manual),
        _ => None,
    }
}

fn parse_timestamp(ts: Option<String>) -> Option<chrono::DateTime<chrono::FixedOffset>> {
    ts.and_then(|s| chrono::DateTime::parse_from_rfc3339(&s).ok())
}

fn map_status(
    status: reqwest::StatusCode,
    body: Result<String, reqwest::Error>,
) -> Result<(), SdzApiError> {
    match status {
        reqwest::StatusCode::OK | reqwest::StatusCode::CREATED => Ok(()),
        reqwest::StatusCode::BAD_REQUEST => {
            tracing::error!("Firestore returned 400: {:?}", body);
            Err(SdzApiError::BadRequest("invalid data for Firestore".into()))
        }
        reqwest::StatusCode::UNAUTHORIZED | reqwest::StatusCode::FORBIDDEN => {
            tracing::error!("Firestore returned auth error: {:?}", body);
            Err(SdzApiError::Unauthorized)
        }
        reqwest::StatusCode::NOT_FOUND => Err(SdzApiError::NotFound),
        code => {
            let text = body.unwrap_or_default();
            tracing::error!("Firestore unexpected status {} body: {}", code, text);
            Err(SdzApiError::Internal)
        }
    }
}

#[derive(Debug, Deserialize)]
struct FirestoreRunQueryResponse {
    document: Option<FirestoreSpotDocWithName>,
}

#[derive(Debug, Deserialize)]
struct FirestoreSpotDocWithName {
    name: String,
    fields: FirestoreSpotFields,
}

impl FirestoreSpotDocWithName {
    fn into_spot(self, spot_id: String) -> SdzSpot {
        let doc = FirestoreSpotDoc {
            fields: self.fields,
        };
        doc.into_spot(spot_id)
    }
}

fn extract_doc_id(name: &str) -> Option<String> {
    name.split('/').next_back().map(|s| s.to_owned())
}
