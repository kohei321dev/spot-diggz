"use strict";

const preferencesStorageKey = "spotdiggz.preferences.v1";

const specifiedLocations = {
  "osaka-station": { label: "大阪駅", latitude: 34.7025, longitude: 135.4960 },
  "namba-station": { label: "なんば駅", latitude: 34.6662, longitude: 135.5001 },
  "sakai-station": { label: "堺駅", latitude: 34.5812, longitude: 135.4683 },
  "nakamozu-station": { label: "なかもず駅", latitude: 34.5567, longitude: 135.5006 },
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
  rail: "レール",
  ledge: "レッジ",
  bank: "バンク",
  "quarter-ramp": "クォーターランプ",
  concrete: "コンクリート路面",
  outdoor: "屋外",
};

const form = document.querySelector("#recommendation-form");
const conditionDetails = document.querySelector("#condition-details");
const conditionSummary = document.querySelector("#condition-summary");
const quickSearchButton = document.querySelector("#quick-search-button");
const moodActionButtons = [...document.querySelectorAll(".mood-action")];
const locateButton = document.querySelector("#locate-button");
const searchButton = document.querySelector("#search-button");
const locationStatus = document.querySelector("#location-status");
const formStatus = document.querySelector("#form-status");
const resultsPanel = document.querySelector(".results-panel");
const results = document.querySelector("#results");
const resultsSummary = document.querySelector("#results-summary");
const resultCount = document.querySelector("#result-count");
const availabilityNotice = document.querySelector("#availability-notice");
const travelNotice = document.querySelector("#travel-notice");
const specifiedLocationFields = document.querySelector("#specified-location-fields");
const currentLocationFields = document.querySelector("#current-location-fields");

let currentLocation = null;
let locationRequest = null;
let isSearching = false;

restorePreferences();
updateOriginMode();
updateConditionSummary();

for (const originMode of document.querySelectorAll('input[name="originMode"]')) {
  originMode.addEventListener("change", updateOriginMode);
}

form.addEventListener("change", updateConditionSummary);
form.addEventListener("submit", (event) => {
  event.preventDefault();
  savePreferences();
  conditionDetails.open = false;
  void searchRecommendations();
});

quickSearchButton.addEventListener("click", () => {
  savePreferences();
  void searchRecommendations();
});

for (const button of moodActionButtons) {
  button.addEventListener("click", () => {
    document.querySelector("#mood").value = button.dataset.mood;
    savePreferences();
    void searchRecommendations();
  });
}

locateButton.addEventListener("click", () => {
  formStatus.textContent = "";
  void requestCurrentLocation().catch((error) => {
    formStatus.textContent = error instanceof Error ? error.message : "現在地を取得できませんでした。";
  });
});

function selectedOriginMode() {
  return document.querySelector('input[name="originMode"]:checked')?.value || "specified_location";
}

function updateOriginMode() {
  const usesCurrentLocation = selectedOriginMode() === "current_location";
  currentLocationFields.hidden = !usesCurrentLocation;
  specifiedLocationFields.hidden = usesCurrentLocation;
  formStatus.textContent = "";
  updateConditionSummary();
}

function updateConditionSummary() {
  const originLabel = selectedOriginMode() === "current_location"
    ? "現在地"
    : specifiedLocations[document.querySelector("#specified-location").value]?.label || "指定地点";
  const labels = [
    originLabel,
    selectedOptionText("#level"),
    selectedOptionText("#transport"),
    selectedOptionText("#available-minutes"),
    selectedOptionText("#purpose"),
  ];
  conditionSummary.textContent = labels.join("・");
}

function selectedOptionText(selector) {
  const select = document.querySelector(selector);
  return select.options[select.selectedIndex]?.textContent || "";
}

function savePreferences() {
  const preferences = {
    originMode: selectedOriginMode(),
    specifiedLocation: document.querySelector("#specified-location").value,
    purpose: document.querySelector("#purpose").value,
    mood: document.querySelector("#mood").value,
    level: document.querySelector("#level").value,
    availableMinutes: document.querySelector("#available-minutes").value,
    transport: document.querySelector("#transport").value,
  };

  try {
    window.localStorage.setItem(preferencesStorageKey, JSON.stringify(preferences));
  } catch {
    // Recommendation still works when browser storage is unavailable.
  }
  updateConditionSummary();
}

function restorePreferences() {
  let preferences;
  try {
    preferences = JSON.parse(window.localStorage.getItem(preferencesStorageKey) || "null");
  } catch {
    return;
  }
  if (!preferences || typeof preferences !== "object") {
    return;
  }

  setRadioValue("originMode", preferences.originMode);
  setSelectValue("#specified-location", preferences.specifiedLocation);
  setSelectValue("#purpose", preferences.purpose);
  setSelectValue("#mood", preferences.mood);
  setSelectValue("#level", preferences.level);
  setSelectValue("#available-minutes", preferences.availableMinutes);
  setSelectValue("#transport", preferences.transport);
}

function setRadioValue(name, value) {
  const radio = document.querySelector(`input[name="${name}"][value="${value}"]`);
  if (radio) {
    radio.checked = true;
  }
}

function setSelectValue(selector, value) {
  const select = document.querySelector(selector);
  if ([...select.options].some((option) => option.value === String(value))) {
    select.value = String(value);
  }
}

async function requestCurrentLocation() {
  if (currentLocation) {
    return currentLocation;
  }
  if (locationRequest) {
    return locationRequest;
  }
  if (!navigator.geolocation) {
    throw new Error("このブラウザでは現在地を取得できません。地点を選んでください。");
  }

  locateButton.disabled = true;
  locationStatus.textContent = "現在地を確認中...";
  locationRequest = new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(
      (position) => {
        currentLocation = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        };
        locationStatus.textContent = "現在地を取得しました。この画面を閉じると破棄されます。";
        resolve(currentLocation);
      },
      () => {
        currentLocation = null;
        locationStatus.textContent = "取得できませんでした。地点を選んで検索できます。";
        reject(new Error("現在地を取得できませんでした。地点を選ぶか、位置情報を許可してください。"));
      },
      { enableHighAccuracy: false, timeout: 10000, maximumAge: 300000 },
    );
  });

  try {
    return await locationRequest;
  } finally {
    locateButton.disabled = false;
    locationRequest = null;
  }
}

async function resolveOrigin() {
  const mode = selectedOriginMode();
  if (mode === "current_location") {
    const location = await requestCurrentLocation();
    return { mode, ...location };
  }

  const location = specifiedLocations[document.querySelector("#specified-location").value];
  return { mode, latitude: location.latitude, longitude: location.longitude };
}

async function searchRecommendations() {
  if (isSearching) {
    return;
  }
  isSearching = true;
  formStatus.textContent = "";
  setLoading(true);

  try {
    const origin = await resolveOrigin();
    const requestBody = {
      purpose: document.querySelector("#purpose").value,
      mood: document.querySelector("#mood").value,
      level: document.querySelector("#level").value,
      availableMinutes: Number(document.querySelector("#available-minutes").value),
      transport: document.querySelector("#transport").value,
      origin,
    };

    const response = await fetch("/api/recommendations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(requestBody),
    });
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body?.error?.message || "おすすめを取得できませんでした。");
    }

    renderRecommendations(body);
    conditionDetails.open = false;
    if (window.matchMedia("(max-width: 760px)").matches) {
      resultsPanel.scrollIntoView({ block: "start", behavior: prefersReducedMotion() ? "auto" : "smooth" });
    }
  } catch (error) {
    results.replaceChildren(createEmptyState("おすすめを取得できませんでした。条件を確認してください。", true));
    resultsSummary.textContent = "検索エラー";
    resultCount.textContent = "ERROR";
    availabilityNotice.hidden = true;
    travelNotice.hidden = true;
    formStatus.textContent = error instanceof Error ? error.message : "おすすめを取得できませんでした。";
  } finally {
    setLoading(false);
    isSearching = false;
  }
}

function setLoading(isLoading) {
  quickSearchButton.disabled = isLoading;
  searchButton.disabled = isLoading;
  for (const button of moodActionButtons) {
    button.disabled = isLoading;
  }
  results.setAttribute("aria-busy", String(isLoading));
  quickSearchButton.textContent = isLoading ? "今日の条件を確認中..." : "おまかせで今から滑る";
  resultsSummary.textContent = isLoading ? "通常営業時間と移動条件を照合しています" : resultsSummary.textContent;
}

function renderRecommendations(body) {
  const recommendations = Array.isArray(body.recommendations) ? body.recommendations : [];
  resultCount.textContent = `${recommendations.length}件`;
  availabilityNotice.textContent = body.availabilityNote || "";
  availabilityNotice.hidden = !availabilityNotice.textContent;
  travelNotice.textContent = body.travelEstimateNote || "";
  travelNotice.hidden = !travelNotice.textContent;

  if (recommendations.length === 0) {
    results.replaceChildren(createEmptyState("今の条件で十分に滑れる候補がありません。時間や出発地点を変えてください。", true));
    resultsSummary.textContent = "該当する候補はありません";
    return;
  }

  const rendered = [createPrimaryRecommendation(recommendations[0])];
  if (recommendations.length > 1) {
    rendered.push(createAlternativeRecommendations(recommendations.slice(1)));
  }
  results.replaceChildren(...rendered);
  resultsSummary.textContent = "通常営業時間内の候補をおすすめ順に選びました";
}

function createEmptyState(message, hasAction = false) {
  const container = document.createElement("div");
  container.className = "empty-state";
  const text = document.createElement("p");
  text.textContent = message;
  container.append(text);

  if (hasAction) {
    const action = document.createElement("button");
    action.className = "empty-action";
    action.type = "button";
    action.textContent = "条件を変更する";
    action.addEventListener("click", () => {
      conditionDetails.open = true;
      conditionDetails.querySelector("summary").focus();
    });
    container.append(action);
  }
  return container;
}

function createPrimaryRecommendation(recommendation) {
  const facility = recommendation.facility;
  const card = document.createElement("article");
  card.className = "result-card primary-result";

  const pickLabel = document.createElement("p");
  pickLabel.className = "pick-label";
  pickLabel.textContent = "今日の一択";

  const title = document.createElement("h3");
  title.className = "result-title";
  title.textContent = facility.name;
  const address = document.createElement("p");
  address.className = "result-address";
  address.textContent = facility.address;

  const timing = document.createElement("div");
  timing.className = "session-timing";
  timing.append(
    createTimingMetric("到着目安", formatClock(recommendation.arrivalAt)),
    createTimingMetric("滑走目安", formatMinutes(recommendation.estimatedSkateMinutes)),
    createTimingMetric("終了目安", formatClock(recommendation.sessionEndsAt)),
  );

  const meta = document.createElement("div");
  meta.className = "result-meta";
  meta.append(createTag(`片道 約${recommendation.estimatedTravelMinutes}分`, "travel"));
  for (const feature of (facility.features || []).slice(0, 4)) {
    meta.append(createTag(featureLabels[feature] || feature));
  }

  const reasons = document.createElement("ul");
  reasons.className = "reasons";
  const primaryReasons = (recommendation.reasons || []).filter(
    (reason) => reason.code !== "travel_estimate" && reason.code !== "session_time_estimate",
  );
  for (const reason of primaryReasons.slice(0, 4)) {
    const item = document.createElement("li");
    item.textContent = reason.message;
    reasons.append(item);
  }

  const details = document.createElement("div");
  details.className = "facility-details";
  details.append(
    createDetail("料金", facility.price || "要確認"),
    createDetail("受付・登録", facility.reservation || "要確認"),
    createDetail("通常営業終了", formatClock(recommendation.facilityClosesAt)),
    createDetail("情報確認日", formatVerifiedAt(facility.verifiedAt)),
  );

  const scheduleNotes = createNotice("営業日の注意", facility.scheduleNotes || []);
  const rules = createRules(facility.rules || []);
  const actions = createPrimaryActions(facility);
  card.append(pickLabel, title, address, timing, meta, reasons, details);
  if (scheduleNotes) {
    card.append(scheduleNotes);
  }
  if (rules) {
    card.append(rules);
  }
  card.append(actions);
  return card;
}

function createTimingMetric(labelText, valueText) {
  const metric = document.createElement("div");
  metric.className = "timing-metric";
  const label = document.createElement("span");
  label.textContent = labelText;
  const value = document.createElement("strong");
  value.textContent = valueText;
  metric.append(label, value);
  return metric;
}

function createRules(rules) {
  return createNotice("利用前に確認", rules);
}

function createNotice(titleText, items) {
  if (items.length === 0) {
    return null;
  }
  const container = document.createElement("div");
  container.className = "rule-notice";
  const heading = document.createElement("p");
  heading.textContent = titleText;
  const list = document.createElement("ul");
  for (const value of items.slice(0, 3)) {
    const item = document.createElement("li");
    item.textContent = value;
    list.append(item);
  }
  container.append(heading, list);
  return container;
}

function createPrimaryActions(facility) {
  const actions = document.createElement("div");
  actions.className = "card-actions";
  const navigationLink = document.createElement("a");
  navigationLink.className = "navigation-link";
  navigationLink.href = buildNavigationURL(facility.location);
  navigationLink.target = "_blank";
  navigationLink.rel = "noopener noreferrer";
  navigationLink.textContent = "このプランで行く";
  const sourceLink = document.createElement("a");
  sourceLink.className = "source-link";
  sourceLink.href = facility.sourceUrl;
  sourceLink.target = "_blank";
  sourceLink.rel = "noopener noreferrer";
  sourceLink.textContent = "情報源と更新日を確認";
  actions.append(navigationLink, sourceLink);
  return actions;
}

function createAlternativeRecommendations(recommendations) {
  const alternatives = document.createElement("details");
  alternatives.className = "alternatives";
  const summary = document.createElement("summary");
  summary.textContent = `ほかの候補 ${recommendations.length}件`;
  const list = document.createElement("div");
  list.className = "alternative-list";

  for (const recommendation of recommendations) {
    const facility = recommendation.facility;
    const row = document.createElement("article");
    row.className = "alternative-row";
    const copy = document.createElement("div");
    const title = document.createElement("h3");
    title.textContent = facility.name;
    const meta = document.createElement("p");
    meta.textContent = `片道 約${recommendation.estimatedTravelMinutes}分・約${formatMinutes(recommendation.estimatedSkateMinutes)}滑走`;
    copy.append(title, meta);
    const link = document.createElement("a");
    link.href = buildNavigationURL(facility.location);
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.textContent = "経路を見る";
    row.append(copy, link);
    list.append(row);
  }

  alternatives.append(summary, list);
  return alternatives;
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

function formatClock(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "要確認";
  }
  return new Intl.DateTimeFormat("ja-JP", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "Asia/Tokyo",
  }).format(date);
}

function formatMinutes(value) {
  const minutes = Number(value);
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return "要確認";
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours === 0) {
    return `${remainingMinutes}分`;
  }
  return remainingMinutes === 0 ? `${hours}時間` : `${hours}時間${remainingMinutes}分`;
}

function buildNavigationURL(location) {
  const destination = `${location.latitude},${location.longitude}`;
  return `https://www.google.com/maps/dir/?api=1&destination=${encodeURIComponent(destination)}`;
}

function prefersReducedMotion() {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
