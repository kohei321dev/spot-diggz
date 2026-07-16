"use strict";

const specifiedLocations = {
  "osaka-station": { latitude: 34.7025, longitude: 135.4960 },
  "namba-station": { latitude: 34.6662, longitude: 135.5001 },
  "sakai-station": { latitude: 34.5812, longitude: 135.4683 },
  "nakamozu-station": { latitude: 34.5567, longitude: 135.5006 },
};

const featureLabels = {
  "flat-area": "フラット",
  indoor: "屋内",
  roof: "屋根あり",
  lighting: "照明",
  rental: "レンタル",
  "mini-ramp": "ミニランプ",
  bowl: "ボウル",
  stairs: "ステア",
  handrail: "ハンドレール",
  outdoor: "屋外",
};

const form = document.querySelector("#recommendation-form");
const locateButton = document.querySelector("#locate-button");
const searchButton = document.querySelector("#search-button");
const locationStatus = document.querySelector("#location-status");
const formStatus = document.querySelector("#form-status");
const results = document.querySelector("#results");
const resultsSummary = document.querySelector("#results-summary");
const resultCount = document.querySelector("#result-count");
const travelNotice = document.querySelector("#travel-notice");
const specifiedLocationFields = document.querySelector("#specified-location-fields");
const currentLocationFields = document.querySelector("#current-location-fields");

let currentLocation = null;

for (const originMode of document.querySelectorAll('input[name="originMode"]')) {
  originMode.addEventListener("change", updateOriginMode);
}

locateButton.addEventListener("click", requestCurrentLocation);
form.addEventListener("submit", searchRecommendations);

function selectedOriginMode() {
  return document.querySelector('input[name="originMode"]:checked').value;
}

function updateOriginMode() {
  const usesCurrentLocation = selectedOriginMode() === "current_location";
  currentLocationFields.hidden = !usesCurrentLocation;
  specifiedLocationFields.hidden = usesCurrentLocation;
  formStatus.textContent = "";
}

function requestCurrentLocation() {
  formStatus.textContent = "";
  if (!navigator.geolocation) {
    locationStatus.textContent = "このブラウザでは現在地を取得できません。";
    return;
  }

  locateButton.disabled = true;
  locationStatus.textContent = "取得中...";
  navigator.geolocation.getCurrentPosition(
    (position) => {
      currentLocation = {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
      };
      locationStatus.textContent = "現在地を取得しました。";
      locateButton.disabled = false;
    },
    () => {
      currentLocation = null;
      locationStatus.textContent = "現在地を取得できませんでした。地点を選んで検索できます。";
      locateButton.disabled = false;
    },
    { enableHighAccuracy: false, timeout: 10000, maximumAge: 300000 },
  );
}

async function searchRecommendations(event) {
  event.preventDefault();
  formStatus.textContent = "";

  const origin = buildOrigin();
  if (!origin) {
    formStatus.textContent = "現在地を取得してから検索してください。";
    return;
  }

  const requestBody = {
    purpose: document.querySelector("#purpose").value,
    mood: document.querySelector("#mood").value,
    level: document.querySelector("#level").value,
    availableMinutes: Number(document.querySelector("#available-minutes").value),
    transport: document.querySelector("#transport").value,
    origin,
  };

  searchButton.disabled = true;
  searchButton.setAttribute("aria-busy", "true");
  searchButton.textContent = "検索中...";
  resultsSummary.textContent = "条件を照合しています";

  try {
    const response = await fetch("/api/recommendations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(requestBody),
    });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body?.error?.message || "検索に失敗しました。");
    }
    renderRecommendations(body);
  } catch (error) {
    results.replaceChildren(createEmptyState("検索結果を取得できませんでした。"));
    resultsSummary.textContent = "検索エラー";
    resultCount.textContent = "0 / 3";
    travelNotice.hidden = true;
    formStatus.textContent = error instanceof Error ? error.message : "検索に失敗しました。";
  } finally {
    searchButton.disabled = false;
    searchButton.removeAttribute("aria-busy");
    searchButton.textContent = "候補を探す";
  }
}

function buildOrigin() {
  const mode = selectedOriginMode();
  if (mode === "current_location") {
    if (!currentLocation) {
      return null;
    }
    return { mode, ...currentLocation };
  }

  const selectedLocation = specifiedLocations[document.querySelector("#specified-location").value];
  return { mode, ...selectedLocation };
}

function renderRecommendations(body) {
  const recommendations = Array.isArray(body.recommendations) ? body.recommendations : [];
  resultCount.textContent = `${recommendations.length} / 3`;
  travelNotice.textContent = body.travelEstimateNote || "";
  travelNotice.hidden = !travelNotice.textContent;

  if (recommendations.length === 0) {
    results.replaceChildren(createEmptyState("条件内で営業中の候補がありません。時間や交通手段を変更してください。"));
    resultsSummary.textContent = "該当する候補はありません";
    return;
  }

  resultsSummary.textContent = `${recommendations.length}件をおすすめ順に表示`;
  results.replaceChildren(
    ...recommendations.map((recommendation, index) => createResultCard(recommendation, index)),
  );
}

function createEmptyState(message) {
  const container = document.createElement("div");
  container.className = "empty-state";
  const text = document.createElement("p");
  text.textContent = message;
  container.append(text);
  return container;
}

function createResultCard(recommendation, index) {
  const facility = recommendation.facility;
  const card = document.createElement("article");
  card.className = "result-card";

  const header = document.createElement("div");
  header.className = "result-card-header";
  const rank = document.createElement("span");
  rank.className = "rank";
  rank.textContent = String(index + 1).padStart(2, "0");
  const titleGroup = document.createElement("div");
  const title = document.createElement("h3");
  title.className = "result-title";
  title.textContent = facility.name;
  const address = document.createElement("p");
  address.className = "result-address";
  address.textContent = facility.address;
  titleGroup.append(title, address);
  header.append(rank, titleGroup);

  const meta = document.createElement("div");
  meta.className = "result-meta";
  meta.append(createTag(`片道 約${recommendation.estimatedTravelMinutes}分`, "travel"));
  meta.append(createTag(`${formatDistance(recommendation.distanceKm)} km`));
  for (const feature of (facility.features || []).slice(0, 3)) {
    meta.append(createTag(featureLabels[feature] || feature));
  }

  const reasons = document.createElement("ul");
  reasons.className = "reasons";
  for (const reason of recommendation.reasons || []) {
    const item = document.createElement("li");
    item.textContent = reason.message;
    reasons.append(item);
  }

  const details = document.createElement("div");
  details.className = "facility-details";
  details.append(
    createDetail("料金", facility.price || "要確認"),
    createDetail("情報確認日", formatVerifiedAt(facility.verifiedAt)),
  );

  const actions = document.createElement("div");
  actions.className = "card-actions";
  const navigationLink = document.createElement("a");
  navigationLink.className = "navigation-link";
  navigationLink.href = buildNavigationURL(facility.location);
  navigationLink.target = "_blank";
  navigationLink.rel = "noopener noreferrer";
  navigationLink.textContent = "地図で経路を確認";
  const sourceLink = document.createElement("a");
  sourceLink.className = "source-link";
  sourceLink.href = facility.sourceUrl;
  sourceLink.target = "_blank";
  sourceLink.rel = "noopener noreferrer";
  sourceLink.textContent = "情報源を見る";
  actions.append(navigationLink, sourceLink);

  card.append(header, meta, reasons, details, actions);
  return card;
}

function createTag(text, modifier = "") {
  const tag = document.createElement("span");
  tag.className = `meta-tag${modifier ? ` ${modifier}` : ""}`;
  tag.textContent = text;
  return tag;
}

function createDetail(labelText, valueText) {
  const container = document.createElement("div");
  const label = document.createElement("span");
  label.className = "detail-label";
  label.textContent = labelText;
  const value = document.createElement("span");
  value.className = "detail-value";
  value.textContent = valueText;
  container.append(label, value);
  return container;
}

function formatVerifiedAt(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "要確認";
  }
  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: "Asia/Tokyo",
  }).format(date);
}

function formatDistance(value) {
  const distance = Number(value);
  return Number.isFinite(distance) ? distance.toFixed(1) : "-";
}

function buildNavigationURL(location) {
  const destination = `${location.latitude},${location.longitude}`;
  return `https://www.google.com/maps/dir/?api=1&destination=${encodeURIComponent(destination)}`;
}
