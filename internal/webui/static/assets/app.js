"use strict";

const localeStorageKey = "spotdiggz.locale.v1";
const defaultLocale = "ja";
const supportedLocales = new Set(["ja", "en"]);
const correctionCategories = new Set(["hours", "price", "rules", "closure", "access", "other"]);
const allowedProductEvents = new Set([
  "input_started",
  "input_completed",
  "recommendation_completed",
  "result_displayed",
  "source_opened",
  "navigation_opened",
  "video_embed_requested",
  "video_embed_loaded",
  "video_external_opened",
  "social_profile_opened",
]);

const messages = {
  ja: {
    documentTitle: "spot-diggz | 今日のスケート先",
    localeLabel: "表示言語",
    brandTagline: "今日の自分に合うスケート先を決める",
    searchTitle: "今から滑る場所を決める",
    searchDescription: "現在の条件を使って、公式情報の通常営業時間内にある候補を選びます。",
    quickActionsLabel: "すぐにおすすめを見る",
    quickSearch: "おまかせで今から滑る",
    quickSearchLoading: "今日の条件を確認中...",
    moodActionsLegend: "今の気分から選ぶ",
    moodEasygoingShort: "気軽に",
    moodEasygoingHint: "近さと使いやすさ",
    moodFocusedShort: "じっくり",
    moodFocusedHint: "練習目的との相性",
    moodChallengeShort: "挑戦する",
    moodChallengeHint: "セクションを重視",
    currentConditions: "現在の条件",
    changeConditions: "条件を変更",
    searchOrigin: "検索位置",
    chooseLocation: "地点を選ぶ",
    currentLocation: "現在地",
    specifiedLocation: "指定地点",
    departurePoint: "出発地点",
    locationSearchLabel: "任意の駅名・住所",
    locationSearchPlaceholder: "例: 京都駅、兵庫県神戸市...",
    locationSearchButton: "検索",
    locationSearchLoadingButton: "検索中...",
    locationSearchProviderNotice: "Google連携が有効な環境では、候補検索のため入力した住所文字列をGoogleへ送る場合があります。外部送信は保存とは別で、住所文字列と候補はこの画面・端末に保存または永続化しません。",
    locationSearchResultsLabel: "検索候補",
    locationSearchRequired: "駅名または住所を入力してください。",
    locationSearchLoading: "候補を検索しています。",
    locationSearchReady: "候補を最大5件表示しました。出発地点を1件選んでください。",
    locationSearchSelectionRequired: "検索候補から出発地点を1件選んでください。",
    locationSearchSelected: "出発地点に「{label}」を選びました。",
    locationSearchEmpty: "一致する候補がありません。別の駅名や住所で検索してください。",
    locationSearchUnavailable: "地点検索は現在利用できません。代表地点または現在地を選んでください。",
    locationSearchFailed: "地点を検索できませんでした。時間をおいて再試行してください。",
    currentLocationProviderNotice: "Google連携が有効な環境では、経路計算のため現在地の座標をGoogleへ送る場合があります。外部送信は保存とは別で、正確な位置はこの画面・端末に保存または永続化しません。",
    locateButton: "現在地を確認",
    purposeLabel: "練習したいこと",
    purposeBasics: "基礎を練習",
    purposeStreet: "ストリートを練習",
    purposeTransition: "ランプ・ボウルを練習",
    moodLabel: "いつもの気分",
    moodFocused: "じっくり練習",
    moodEasygoing: "気軽に滑る",
    moodChallenge: "新しいことに挑戦",
    levelLabel: "レベル",
    levelBeginner: "初心者",
    levelReturning: "久しぶり",
    levelIntermediate: "中級",
    availableTimeLabel: "使える時間",
    duration60: "1時間",
    duration120: "2時間",
    duration180: "3時間",
    duration240: "4時間",
    transportLabel: "交通手段",
    transportPublic: "公共交通",
    transportCar: "車",
    transportBicycle: "自転車",
    transportWalk: "徒歩",
    preferencePrivacy: "表示言語だけをこの端末に保存します。正確な現在地と検索条件は保存しません。",
    updateRecommendations: "この条件でおすすめを更新",
    resultsTitle: "今日のおすすめ",
    initialResultsSummary: "「今から滑る」で今日の一択を表示します",
    initialEmpty: "条件入力は不要です。現在の設定ですぐに探せます。",
    loadingResultsSummary: "通常営業時間と移動条件を照合しています",
    loadingEmpty: "候補を確認しています。",
    errorResultsSummary: "検索エラー",
    recommendationError: "おすすめを取得できませんでした。時間をおいて再試行してください。",
    recommendationErrorEmpty: "おすすめを取得できませんでした。条件を確認して再試行してください。",
    emptyResultsSummary: "該当する候補はありません",
    emptyResults: "今の条件で十分に滑れる候補がありません。出発地点・交通手段・利用可能時間を変えるか、営業時間内に再試行してください。",
    successResultsSummary: "通常営業時間内の候補をおすすめ順に選びました",
    resultCount: "{count}件",
    ready: "READY",
    error: "ERROR",
    changeConditionsAction: "条件を変更する",
    locationIdle: "検索時に取得します",
    locationLoading: "現在地を確認中...",
    locationSuccess: "現在地を取得しました。この画面を閉じると破棄されます。",
    locationFailure: "取得できませんでした。地点を選んで検索できます。",
    locationUnsupported: "このブラウザでは現在地を取得できません。地点を選んでください。",
    locationPermissionError: "現在地を取得できませんでした。地点を選ぶか、位置情報を許可してください。",
    availabilityNotice: "表示は公式情報の通常営業時間に基づきます。臨時休場、貸切、天候による変更は公式情報で確認してください。",
    travelNotice: "移動時間は候補ごとの表示種別を確認し、実際の経路と所要時間は外部ナビで確認してください。",
    travelNoticeStraight: "移動時間は直線距離による概算です。実際の経路と所要時間は外部ナビで確認してください。",
    travelNoticeGoogle: "移動時間はGoogle Maps Routes APIの実経路に基づく目安です。運休、渋滞、経路変更は外部ナビで確認してください。",
    travelNoticeMixed: "Google実経路の目安と直線距離による概算が混在しています。候補ごとの表示種別と外部ナビを確認してください。",
    travelKindStraight: "直線距離の概算",
    travelKindGoogle: "Google実経路",
    primaryPick: "今日の一択",
    arrivalEstimate: "到着目安",
    skateEstimate: "滑走目安",
    sessionEndEstimate: "終了目安",
    oneWayTravel: "片道 約{minutes}分",
    alternativeTiming: "片道 約{travel}分・約{skate}滑走",
    price: "料金",
    reservation: "受付・登録",
    closingTime: "通常営業終了",
    verifiedAt: "情報確認日",
    accessNotes: "アクセス補足",
    needsConfirmation: "要確認",
    scheduleCaution: "営業日の注意",
    availabilityCaution: "一般利用の注意",
    rulesCaution: "利用前に確認",
    goWithPlan: "このプランで行く",
    routeLink: "経路を見る",
    officialSource: "公式情報",
    reportCorrection: "情報の誤りを報告",
    mediaSection: "動画・公式SNS",
    youtubeVideo: "YouTube動画",
    youtubePrivacyNotice: "動画を表示するとYouTubeに接続します。自動再生はしません。",
    showVideo: "動画を表示",
    openYouTube: "YouTubeで開く",
    openInstagram: "Instagramを開く",
    openX: "Xを開く",
    translationFallback: "英語情報が未整備のため日本語を表示しています。来場前に公式情報を確認してください。",
    alternatives: "ほかの候補 {count}件",
    alternativeReasonLabel: "おすすめ理由",
    reasonPurposeBasics: "基礎練習に合うフラットエリアがあります。",
    reasonPurposeStreet: "ストリート練習に合うセクションがあります。",
    reasonPurposeTransition: "ランプやボウルの練習に合う設備があります。",
    reasonBeginner: "初心者向けとして確認されている施設です。",
    reasonMoodFocused: "目的の練習に集中しやすい設備があります。",
    reasonMoodEasygoing: "気軽に滑りやすい設備条件があります。",
    reasonMoodChallenge: "挑戦向けのセクションがあります。",
    reasonSessionTime: "往復の概算移動と通常営業時間を考慮すると約{minutes}分滑れる見込みです。",
    reasonTravel: "選択した交通手段で片道約{minutes}分の概算です。",
    reasonFallback: "選択した条件に合う候補です。",
    correctionTitle: "情報の誤りを報告",
    closeDialog: "閉じる",
    correctionCategory: "訂正する項目",
    requiredLabel: "必須",
    optionalLabel: "任意",
    categoryHours: "営業時間",
    categoryPrice: "料金",
    categoryRules: "利用ルール",
    categoryClosure: "休場・閉鎖",
    categoryAccess: "アクセス",
    categoryOther: "その他",
    correctionDetails: "訂正内容",
    correctionDetailsPlaceholder: "誤っている情報と、確認できた正しい情報を10文字以上で入力してください",
    correctionDetailsHelp: "10〜1000文字で入力してください。",
    evidenceUrlLabel: "根拠URL",
    evidenceUrlHelp: "施設や自治体の公式ページがあれば入力してください。",
    contactLabel: "連絡先メールアドレス",
    contactUseNote: "確認が必要な場合の連絡だけに利用し、送信から90日を上限に保持して定期的に削除します。",
    contactConsentLabel: "連絡先の利用目的と、送信から最長90日間の保持・定期削除に同意します。",
    correctionPrivacyWarning: "訂正内容や根拠URLに、正確な現在地、氏名、電話番号などの個人情報を書かないでください。",
    cancel: "キャンセル",
    submitCorrection: "報告を送信",
    submittingCorrection: "送信中...",
    retryCorrection: "もう一度送信",
    correctionSendingStatus: "訂正報告を送信しています。",
    correctionSubmitFailed: "送信できませんでした。入力内容は保持されています。もう一度送信してください。",
    correctionServerValidation: "入力内容を確認できませんでした。各項目を見直して、もう一度送信してください。",
    correctionAccepted: "報告を受け付けました",
    reportIdLabel: "受付番号",
    correctionNextAction: "施設の公式情報と照合し、必要な場合はカタログを更新します。",
    validationCategory: "訂正する項目を選んでください。",
    validationDetailsRequired: "訂正内容を入力してください。",
    validationDetailsLength: "訂正内容は10〜1000文字で入力してください。",
    validationEvidenceUrl: "根拠URLは https:// から始まる有効なURLを入力してください。",
    validationContact: "連絡先は有効なメールアドレスを入力してください。",
    validationConsent: "連絡先を入力する場合は、利用目的と最長90日間の保持・定期削除への同意が必要です。",
  },
  en: {
    documentTitle: "spot-diggz | Today's skate spot",
    localeLabel: "Display language",
    brandTagline: "Choose the skate spot that fits today",
    searchTitle: "Choose where to skate now",
    searchDescription: "We'll use your current choices to find spots within their published regular hours.",
    quickActionsLabel: "Get a recommendation now",
    quickSearch: "Pick a spot for me",
    quickSearchLoading: "Checking today's options...",
    moodActionsLegend: "Choose by mood",
    moodEasygoingShort: "Easygoing",
    moodEasygoingHint: "Closer and easier to use",
    moodFocusedShort: "Focused",
    moodFocusedHint: "A fit for your practice goal",
    moodChallengeShort: "Challenge",
    moodChallengeHint: "More sections to try",
    currentConditions: "Current choices",
    changeConditions: "Change conditions",
    searchOrigin: "Starting point",
    chooseLocation: "Choose a station",
    currentLocation: "Current location",
    specifiedLocation: "Selected location",
    departurePoint: "Departure point",
    locationSearchLabel: "Any station or address",
    locationSearchPlaceholder: "Example: Kyoto Station or an address in Kobe",
    locationSearchButton: "Search",
    locationSearchLoadingButton: "Searching...",
    locationSearchProviderNotice: "When Google integration is enabled, the address text you enter may be sent to Google to find location candidates. This external transfer is separate from storage; the address text and candidates are not saved or persisted on this screen or device.",
    locationSearchResultsLabel: "Location search results",
    locationSearchRequired: "Enter a station name or address.",
    locationSearchLoading: "Searching for locations.",
    locationSearchReady: "Up to five results are shown. Select one as your departure point.",
    locationSearchSelectionRequired: "Select one departure point from the search results.",
    locationSearchSelected: "“{label}” is selected as your departure point.",
    locationSearchEmpty: "No matching location was found. Try another station name or address.",
    locationSearchUnavailable: "Location search is currently unavailable. Choose a listed station or use your current location.",
    locationSearchFailed: "We couldn't search for that location. Please try again shortly.",
    currentLocationProviderNotice: "When Google integration is enabled, your current coordinates may be sent to Google to calculate routes. This external transfer is separate from storage; your precise location is not saved or persisted on this screen or device.",
    locateButton: "Use current location",
    purposeLabel: "Practice goal",
    purposeBasics: "Practice basics",
    purposeStreet: "Practice street",
    purposeTransition: "Practice ramps and bowls",
    moodLabel: "Usual mood",
    moodFocused: "Focused practice",
    moodEasygoing: "Easygoing session",
    moodChallenge: "Try something new",
    levelLabel: "Level",
    levelBeginner: "Beginner",
    levelReturning: "Returning",
    levelIntermediate: "Intermediate",
    availableTimeLabel: "Time available",
    duration60: "1 hour",
    duration120: "2 hours",
    duration180: "3 hours",
    duration240: "4 hours",
    transportLabel: "Transport",
    transportPublic: "Public transit",
    transportCar: "Car",
    transportBicycle: "Bicycle",
    transportWalk: "Walk",
    preferencePrivacy: "Only your display language is saved on this device. Your precise location and search choices are not saved.",
    updateRecommendations: "Update recommendations",
    resultsTitle: "Today's recommendations",
    initialResultsSummary: "Select “Pick a spot for me” to see today's top pick",
    initialEmpty: "No setup is required. Search now with the current choices.",
    loadingResultsSummary: "Checking regular hours and travel conditions",
    loadingEmpty: "Checking available spots.",
    errorResultsSummary: "Search error",
    recommendationError: "We couldn't get recommendations. Please try again shortly.",
    recommendationErrorEmpty: "We couldn't get recommendations. Check the conditions and try again.",
    emptyResultsSummary: "No matching spots",
    emptyResults: "No spot leaves enough time to skate with these conditions. Change the departure point, transport, or available time, or try again during opening hours.",
    successResultsSummary: "Ranked spots that fit their published regular hours",
    resultCount: "{count}",
    ready: "READY",
    error: "ERROR",
    changeConditionsAction: "Change conditions",
    locationIdle: "Requested when you search",
    locationLoading: "Checking your current location...",
    locationSuccess: "Location received. It will be discarded when you close this page.",
    locationFailure: "Location unavailable. You can search from a station instead.",
    locationUnsupported: "This browser cannot provide your location. Choose a station instead.",
    locationPermissionError: "We couldn't access your location. Choose a station or allow location access.",
    availabilityNotice: "Results use published regular hours. Check the official source for temporary closures, private bookings, and weather changes.",
    travelNotice: "Check each result's travel estimate type, then confirm the actual route and duration in external navigation.",
    travelNoticeStraight: "Travel times are straight-line estimates, not actual routes. Confirm the route and duration in external navigation.",
    travelNoticeGoogle: "Travel times use Google Maps Routes API route results. Confirm service changes, traffic, and the current route in external navigation.",
    travelNoticeMixed: "Results include both Google route estimates and straight-line estimates. Check each result's label and confirm the route in external navigation.",
    travelKindStraight: "Straight-line estimate",
    travelKindGoogle: "Google route",
    primaryPick: "TODAY'S TOP PICK",
    arrivalEstimate: "Arrival",
    skateEstimate: "Skate time",
    sessionEndEstimate: "Leave by",
    oneWayTravel: "About {minutes} min one way",
    alternativeTiming: "About {travel} min one way · about {skate} skating",
    price: "Price",
    reservation: "Registration",
    closingTime: "Regular closing",
    verifiedAt: "Information checked",
    accessNotes: "Access note",
    needsConfirmation: "Check official source",
    scheduleCaution: "Schedule notes",
    availabilityCaution: "General-use availability",
    rulesCaution: "Check before visiting",
    goWithPlan: "Navigate to this spot",
    routeLink: "View route",
    officialSource: "Official source",
    reportCorrection: "Report incorrect information",
    mediaSection: "Video and official social profiles",
    youtubeVideo: "YouTube video",
    youtubePrivacyNotice: "Showing this video connects to YouTube. It will not autoplay.",
    showVideo: "Show video",
    openYouTube: "Open on YouTube",
    openInstagram: "Open Instagram",
    openX: "Open X",
    translationFallback: "English details are not available yet, so the Japanese facility information is shown. Check the official source before visiting.",
    alternatives: "{count} alternatives",
    alternativeReasonLabel: "Why it fits",
    reasonPurposeBasics: "It has a flat area suited to practicing basics.",
    reasonPurposeStreet: "It has sections suited to street practice.",
    reasonPurposeTransition: "It has ramps or bowls suited to transition practice.",
    reasonBeginner: "This spot is confirmed as beginner-friendly.",
    reasonMoodFocused: "Its facilities support focused practice for your goal.",
    reasonMoodEasygoing: "It has features that make an easygoing session more practical.",
    reasonMoodChallenge: "It has sections suited to trying new challenges.",
    reasonSessionTime: "After estimated return travel and regular hours, you should have about {minutes} minutes to skate.",
    reasonTravel: "The estimated one-way trip is about {minutes} minutes with your selected transport.",
    reasonFallback: "This spot matches the selected conditions.",
    correctionTitle: "Report incorrect information",
    closeDialog: "Close",
    correctionCategory: "Information to correct",
    requiredLabel: "Required",
    optionalLabel: "Optional",
    categoryHours: "Hours",
    categoryPrice: "Price",
    categoryRules: "Rules",
    categoryClosure: "Closure",
    categoryAccess: "Access",
    categoryOther: "Other",
    correctionDetails: "Correction details",
    correctionDetailsPlaceholder: "In at least 10 characters, describe what is wrong and the correct information you found",
    correctionDetailsHelp: "Enter 10 to 1,000 characters.",
    evidenceUrlLabel: "Evidence URL",
    evidenceUrlHelp: "Add an official facility or municipal page when available.",
    contactLabel: "Contact email",
    contactUseNote: "Used only if clarification is needed, retained for no more than 90 days after submission, and deleted on a recurring schedule.",
    contactConsentLabel: "I consent to this contact purpose, retention for up to 90 days after submission, and periodic deletion.",
    correctionPrivacyWarning: "Do not include your precise location, name, phone number, or other personal information in the details or evidence URL.",
    cancel: "Cancel",
    submitCorrection: "Submit report",
    submittingCorrection: "Submitting...",
    retryCorrection: "Try again",
    correctionSendingStatus: "Submitting your correction report.",
    correctionSubmitFailed: "The report could not be submitted. Your entries are still here; please try again.",
    correctionServerValidation: "The submission could not be validated. Review each field and try again.",
    correctionAccepted: "Report received",
    reportIdLabel: "Report ID",
    correctionNextAction: "We will compare the report with the facility's official information and update the catalog when needed.",
    validationCategory: "Choose the information category.",
    validationDetailsRequired: "Enter the correction details.",
    validationDetailsLength: "Correction details must be 10 to 1,000 characters.",
    validationEvidenceUrl: "Enter a valid evidence URL beginning with https://.",
    validationContact: "Enter a valid contact email address.",
    validationConsent: "A contact email requires consent to its purpose, retention for up to 90 days, and periodic deletion.",
  },
};

const specifiedLocations = {
  "osaka-station": {
    labels: { ja: "大阪駅", en: "Osaka Station" },
    latitude: 34.7025,
    longitude: 135.4960,
  },
  "namba-station": {
    labels: { ja: "なんば駅", en: "Namba Station" },
    latitude: 34.6662,
    longitude: 135.5001,
  },
  "sakai-station": {
    labels: { ja: "堺駅", en: "Sakai Station" },
    latitude: 34.5812,
    longitude: 135.4683,
  },
  "nakamozu-station": {
    labels: { ja: "なかもず駅", en: "Nakamozu Station" },
    latitude: 34.5567,
    longitude: 135.5006,
  },
  "kobe-station": {
    labels: { ja: "神戸駅", en: "Kobe Station" },
    latitude: 34.6796,
    longitude: 135.1780,
  },
  "himeji-station": {
    labels: { ja: "姫路駅", en: "Himeji Station" },
    latitude: 34.8275,
    longitude: 134.6908,
  },
  "wakayama-station": {
    labels: { ja: "和歌山駅", en: "Wakayama Station" },
    latitude: 34.2320,
    longitude: 135.1910,
  },
  "nara-station": {
    labels: { ja: "奈良駅", en: "Nara Station" },
    latitude: 34.6805,
    longitude: 135.8186,
  },
  "tokushima-station": {
    labels: { ja: "徳島駅", en: "Tokushima Station" },
    latitude: 34.0742,
    longitude: 134.5514,
  },
};

const featureLabels = {
  ja: {
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
    rooftop: "屋上",
  },
  en: {
    "flat-area": "Flat area",
    indoor: "Indoor",
    roof: "Covered",
    lighting: "Lighting",
    rental: "Rental",
    "mini-ramp": "Mini ramp",
    bowl: "Bowl",
    stairs: "Stairs",
    handrail: "Handrail",
    rail: "Rail",
    ledge: "Ledge",
    bank: "Bank",
    "quarter-ramp": "Quarter pipe",
    concrete: "Concrete",
    outdoor: "Outdoor",
    rooftop: "Rooftop",
  },
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
const specifiedLocationSelect = document.querySelector("#specified-location");
const locationSearchQuery = document.querySelector("#location-search-query");
const locationSearchButton = document.querySelector("#location-search-button");
const locationSearchStatus = document.querySelector("#location-search-status");
const locationSearchResults = document.querySelector("#location-search-results");
const correctionDialog = document.querySelector("#correction-dialog");
const correctionForm = document.querySelector("#correction-form");
const correctionFacility = document.querySelector("#correction-facility");
const correctionFacilityID = document.querySelector("#correction-facility-id");
const correctionCategory = document.querySelector("#correction-category");
const correctionDetails = document.querySelector("#correction-details");
const correctionEvidenceURL = document.querySelector("#correction-evidence-url");
const correctionContact = document.querySelector("#correction-contact");
const correctionContactConsent = document.querySelector("#correction-contact-consent");
const correctionStatus = document.querySelector("#correction-status");
const correctionSubmitButton = document.querySelector("#correction-submit-button");
const correctionSuccess = document.querySelector("#correction-success");
const correctionReportID = document.querySelector("#correction-report-id");
const correctionNextAction = document.querySelector("#correction-next-action");

let currentLocale = restoreLocale();
let currentLocation = null;
let locationRequest = null;
let locationState = "idle";
let searchedLocations = [];
let selectedSearchedLocation = null;
let locationSearchState = "idle";
let locationSearchController = null;
let formStatusKey = "";
let hasRecordedInputStarted = false;
let isSearching = false;
let resultState = "idle";
let lastRecommendationBody = null;
let lastSearchContext = null;
let activeCorrectionFacility = null;
let correctionTrigger = null;
let isCorrectionSubmitting = false;
let correctionRequestVersion = 0;
let correctionState = { type: "idle", messageKey: "", reportId: "", nextAction: "" };

applyLocale(currentLocale, false);
updateOriginMode();
renderResults();

for (const localeButton of document.querySelectorAll("[data-locale]")) {
  localeButton.addEventListener("click", () => applyLocale(localeButton.dataset.locale, true));
}

for (const originMode of document.querySelectorAll('input[name="originMode"]')) {
  originMode.addEventListener("change", updateOriginMode);
}

specifiedLocationSelect.addEventListener("change", () => {
  clearSearchedLocation(true);
  updateConditionSummary();
});

locationSearchButton.addEventListener("click", () => {
  void searchLocations();
});

locationSearchQuery.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    void searchLocations();
  }
});

locationSearchQuery.addEventListener("input", () => {
  clearSearchedLocation(false);
  if (formStatusKey === "locationSearchSelectionRequired") {
    setFormStatus("");
  }
  updateConditionSummary();
});

conditionDetails.querySelector("summary").addEventListener("click", recordInputStarted);
form.addEventListener("input", recordInputStarted);
form.addEventListener("change", () => {
  recordInputStarted();
  updateConditionSummary();
});
form.addEventListener("submit", (event) => {
  event.preventDefault();
  void searchRecommendations();
});

quickSearchButton.addEventListener("click", () => {
  void searchRecommendations();
});

for (const button of moodActionButtons) {
  button.addEventListener("click", () => {
    recordInputStarted();
    document.querySelector("#mood").value = button.dataset.mood;
    updateConditionSummary();
    void searchRecommendations();
  });
}

locateButton.addEventListener("click", () => {
  setFormStatus("");
  void requestCurrentLocation().catch((error) => {
    showSpecifiedLocationFallback(
      error instanceof LocalizedError ? error.messageKey : "locationPermissionError",
    );
  });
});

document.querySelector("#correction-close-button").addEventListener("click", closeCorrectionDialog);
document.querySelector("#correction-cancel-button").addEventListener("click", closeCorrectionDialog);
document.querySelector("#correction-success-close-button").addEventListener("click", closeCorrectionDialog);

correctionDialog.addEventListener("click", (event) => {
  if (event.target === correctionDialog) {
    closeCorrectionDialog();
  }
});

correctionDialog.addEventListener("close", () => {
  const trigger = correctionTrigger;
  correctionTrigger = null;
  activeCorrectionFacility = null;
  if (trigger?.isConnected) {
    trigger.focus();
  }
});

correctionContact.addEventListener("input", updateContactConsentState);
correctionForm.addEventListener("submit", (event) => {
  event.preventDefault();
  void submitCorrection();
});

class LocalizedError extends Error {
  constructor(messageKey) {
    super(messageKey);
    this.messageKey = messageKey;
  }
}

function t(key, replacements = {}) {
  const template = messages[currentLocale]?.[key] ?? messages[defaultLocale]?.[key] ?? key;
  return Object.entries(replacements).reduce(
    (message, [name, value]) => message.replaceAll(`{${name}}`, String(value)),
    template,
  );
}

function restoreLocale() {
  try {
    const storedLocale = window.localStorage.getItem(localeStorageKey);
    if (supportedLocales.has(storedLocale)) {
      return storedLocale;
    }
  } catch {
    // Language switching still works for the current page when storage is unavailable.
  }
  return defaultLocale;
}

function applyLocale(locale, shouldPersist) {
  if (!supportedLocales.has(locale)) {
    return;
  }
  currentLocale = locale;
  document.documentElement.lang = locale;
  document.title = t("documentTitle");

  for (const element of document.querySelectorAll("[data-i18n]")) {
    element.textContent = t(element.dataset.i18n);
  }
  for (const element of document.querySelectorAll("[data-i18n-aria-label]")) {
    element.setAttribute("aria-label", t(element.dataset.i18nAriaLabel));
  }
  for (const element of document.querySelectorAll("[data-i18n-title]")) {
    element.title = t(element.dataset.i18nTitle);
  }
  for (const element of document.querySelectorAll("[data-i18n-placeholder]")) {
    element.placeholder = t(element.dataset.i18nPlaceholder);
  }
  for (const option of document.querySelectorAll("[data-option-key]")) {
    option.textContent = t(option.dataset.optionKey);
  }
  for (const option of document.querySelectorAll("[data-location-key]")) {
    const location = specifiedLocations[option.dataset.locationKey];
    option.textContent = location?.labels[currentLocale] || option.textContent;
  }
  for (const button of document.querySelectorAll("[data-locale]")) {
    button.setAttribute("aria-pressed", String(button.dataset.locale === currentLocale));
  }

  if (shouldPersist) {
    try {
      window.localStorage.setItem(localeStorageKey, currentLocale);
    } catch {
      // The selected language remains active for the current page.
    }
  }

  updateConditionSummary();
  renderLocationStatus();
  renderLocationSearch();
  renderFormStatus();
  renderSearchControls();
  renderResults();
  renderCorrectionState();
}

function selectedOriginMode() {
  return document.querySelector('input[name="originMode"]:checked')?.value || "specified_location";
}

function updateOriginMode() {
  const usesCurrentLocation = selectedOriginMode() === "current_location";
  currentLocationFields.hidden = !usesCurrentLocation;
  specifiedLocationFields.hidden = usesCurrentLocation;
  setFormStatus("");
  updateConditionSummary();
}

function updateConditionSummary() {
  const selectedLocationKey = specifiedLocationSelect.value;
  const originLabel = selectedOriginMode() === "current_location"
    ? t("currentLocation")
    : selectedSearchedLocation?.label
      || specifiedLocations[selectedLocationKey]?.labels[currentLocale]
      || t("specifiedLocation");
  const labels = [
    originLabel,
    selectedOptionText("#level"),
    selectedOptionText("#transport"),
    selectedOptionText("#available-minutes"),
    selectedOptionText("#purpose"),
  ];
  conditionSummary.textContent = labels.join(currentLocale === "ja" ? "・" : " · ");
}

function selectedOptionText(selector) {
  const select = document.querySelector(selector);
  return select.options[select.selectedIndex]?.textContent || "";
}

async function requestCurrentLocation() {
  if (currentLocation) {
    locationState = "success";
    renderLocationStatus();
    return currentLocation;
  }
  if (locationRequest) {
    return locationRequest;
  }
  if (!navigator.geolocation) {
    locationState = "unsupported";
    renderLocationStatus();
    throw new LocalizedError("locationUnsupported");
  }

  locateButton.disabled = true;
  locationState = "loading";
  renderLocationStatus();
  locationRequest = new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(
      (position) => {
        currentLocation = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
        };
        locationState = "success";
        renderLocationStatus();
        resolve(currentLocation);
      },
      () => {
        currentLocation = null;
        locationState = "failure";
        renderLocationStatus();
        reject(new LocalizedError("locationPermissionError"));
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

function renderLocationStatus() {
  const stateKeys = {
    idle: "locationIdle",
    loading: "locationLoading",
    success: "locationSuccess",
    failure: "locationFailure",
    unsupported: "locationUnsupported",
  };
  locationStatus.textContent = t(stateKeys[locationState] || "locationIdle");
}

async function searchLocations() {
  const query = locationSearchQuery.value.trim();
  if (formStatusKey === "locationSearchSelectionRequired") {
    setFormStatus("");
  }
  if (!query) {
    searchedLocations = [];
    selectedSearchedLocation = null;
    locationSearchState = "required";
    renderLocationSearch();
    updateConditionSummary();
    locationSearchQuery.focus();
    return;
  }

  locationSearchController?.abort();
  const controller = new AbortController();
  locationSearchController = controller;
  searchedLocations = [];
  selectedSearchedLocation = null;
  locationSearchState = "loading";
  renderLocationSearch();
  updateConditionSummary();

  try {
    const response = await fetch("/api/locations/search", {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ query }),
      signal: controller.signal,
    });
    if (response.status === 503) {
      locationSearchState = "unavailable";
      return;
    }
    const body = await parseJSON(response);
    if (!response.ok) {
      locationSearchState = "failed";
      return;
    }

    searchedLocations = extractLocationResults(body).slice(0, 5);
    locationSearchState = searchedLocations.length > 0 ? "ready" : "empty";
  } catch (error) {
    if (error?.name === "AbortError") {
      return;
    }
    locationSearchState = "failed";
  } finally {
    if (locationSearchController === controller) {
      locationSearchController = null;
      renderLocationSearch();
    }
  }
}

function extractLocationResults(body) {
  const values = Array.isArray(body)
    ? body
    : Array.isArray(body?.locations)
      ? body.locations
      : Array.isArray(body?.results)
        ? body.results
        : [];
  return values.flatMap((value) => {
    const latitude = Number(value?.location?.latitude);
    const longitude = Number(value?.location?.longitude);
    if (typeof value?.label !== "string" || !value.label.trim()
      || !Number.isFinite(latitude) || latitude < -90 || latitude > 90
      || !Number.isFinite(longitude) || longitude < -180 || longitude > 180) {
      return [];
    }
    return [{
      label: value.label.trim(),
      location: { latitude, longitude },
    }];
  });
}

function renderLocationSearch() {
  const statusKeys = {
    required: "locationSearchRequired",
    loading: "locationSearchLoading",
    ready: "locationSearchReady",
    selection_required: "locationSearchSelectionRequired",
    empty: "locationSearchEmpty",
    unavailable: "locationSearchUnavailable",
    failed: "locationSearchFailed",
  };
  locationSearchButton.disabled = locationSearchState === "loading";
  locationSearchButton.textContent = t(locationSearchState === "loading"
    ? "locationSearchLoadingButton"
    : "locationSearchButton");
  locationSearchStatus.textContent = locationSearchState === "selected" && selectedSearchedLocation
    ? t("locationSearchSelected", { label: selectedSearchedLocation.label })
    : statusKeys[locationSearchState]
      ? t(statusKeys[locationSearchState])
      : "";
  locationSearchStatus.classList.toggle(
    "is-error",
    ["required", "selection_required", "unavailable", "failed"].includes(locationSearchState),
  );
  locationSearchResults.setAttribute("aria-busy", String(locationSearchState === "loading"));
  locationSearchResults.replaceChildren();

  for (const [index, result] of searchedLocations.entries()) {
    const option = document.createElement("label");
    option.className = "location-search-result";
    const radio = document.createElement("input");
    radio.type = "radio";
    radio.name = "searchedLocation";
    radio.value = String(index);
    radio.checked = selectedSearchedLocation === result;
    const label = document.createElement("span");
    label.textContent = result.label;
    radio.addEventListener("change", () => {
      selectedSearchedLocation = result;
      locationSearchState = "selected";
      locationSearchStatus.textContent = t("locationSearchSelected", { label: result.label });
      locationSearchStatus.classList.remove("is-error");
      if (formStatusKey === "locationSearchSelectionRequired") {
        setFormStatus("");
      }
      updateConditionSummary();
    });
    option.append(radio, label);
    locationSearchResults.append(option);
  }
  locationSearchResults.hidden = searchedLocations.length === 0;
}

function clearSearchedLocation(shouldClearQuery = false) {
  locationSearchController?.abort();
  locationSearchController = null;
  searchedLocations = [];
  selectedSearchedLocation = null;
  locationSearchState = "idle";
  if (shouldClearQuery) {
    locationSearchQuery.value = "";
  }
  renderLocationSearch();
}

function showLocationSelectionError() {
  locationSearchState = "selection_required";
  conditionDetails.open = true;
  renderLocationSearch();
  const firstCandidate = locationSearchResults.querySelector('input[name="searchedLocation"]');
  (firstCandidate || locationSearchQuery).focus();
}

function showSpecifiedLocationFallback(messageKey) {
  const specifiedLocationMode = document.querySelector(
    'input[name="originMode"][value="specified_location"]',
  );
  specifiedLocationMode.checked = true;
  conditionDetails.open = true;
  updateOriginMode();
  setFormStatus(messageKey);
  specifiedLocationSelect.focus();
}

async function resolveOrigin() {
  const mode = selectedOriginMode();
  if (mode === "current_location") {
    const location = await requestCurrentLocation();
    return { mode, ...location };
  }

  if (locationSearchQuery.value.trim() && !selectedSearchedLocation) {
    showLocationSelectionError();
    throw new LocalizedError("locationSearchSelectionRequired");
  }

  const selectedLocation = selectedSearchedLocation?.location
    || specifiedLocations[specifiedLocationSelect.value]
    || specifiedLocations["osaka-station"];
  return {
    mode,
    latitude: selectedLocation.latitude,
    longitude: selectedLocation.longitude,
  };
}

async function searchRecommendations() {
  if (isSearching) {
    return;
  }

  isSearching = true;
  resultState = "loading";
  setFormStatus("");
  renderSearchControls();
  renderResults();

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
    lastSearchContext = {
      ...requestBody,
      origin: origin.mode === "specified_location" ? { ...origin } : { mode: origin.mode },
    };
    recordProductEvent("input_completed");

    const response = await fetch("/api/recommendations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(requestBody),
    });
    const body = await parseJSON(response);
    if (!response.ok || !body || !Array.isArray(body.recommendations)) {
      throw new LocalizedError("recommendationError");
    }
    recordProductEvent("recommendation_completed");

    lastRecommendationBody = body;
    resultState = body.recommendations.length > 0 ? "success" : "empty";
    conditionDetails.open = false;
    renderResults();
    recordProductEvent("result_displayed");
    if (window.matchMedia("(max-width: 760px)").matches) {
      resultsPanel.scrollIntoView({ block: "start", behavior: prefersReducedMotion() ? "auto" : "smooth" });
    }
  } catch (error) {
    lastRecommendationBody = null;
    resultState = "error";
    const messageKey = error instanceof LocalizedError ? error.messageKey : "recommendationError";
    if (messageKey === "locationSearchSelectionRequired") {
      showLocationSelectionError();
      setFormStatus(messageKey);
    } else if (messageKey === "locationPermissionError" || messageKey === "locationUnsupported") {
      showSpecifiedLocationFallback(messageKey);
    } else {
      setFormStatus(messageKey);
    }
    renderResults();
  } finally {
    isSearching = false;
    renderSearchControls();
  }
}

function renderSearchControls() {
  quickSearchButton.disabled = isSearching;
  searchButton.disabled = isSearching;
  locateButton.disabled = locationState === "loading";
  for (const button of moodActionButtons) {
    button.disabled = isSearching;
  }
  quickSearchButton.textContent = t(isSearching ? "quickSearchLoading" : "quickSearch");
  searchButton.textContent = t("updateRecommendations");
}

function renderResults() {
  results.setAttribute("aria-busy", String(resultState === "loading"));
  availabilityNotice.hidden = true;
  travelNotice.hidden = true;

  switch (resultState) {
    case "loading":
      results.replaceChildren(createEmptyState(t("loadingEmpty")));
      resultsSummary.textContent = t("loadingResultsSummary");
      resultCount.textContent = "...";
      return;
    case "error":
      results.replaceChildren(createEmptyState(t("recommendationErrorEmpty"), true));
      resultsSummary.textContent = t("errorResultsSummary");
      resultCount.textContent = t("error");
      return;
    case "success":
    case "empty":
      renderRecommendationResponse(lastRecommendationBody || { recommendations: [] });
      return;
    default:
      results.replaceChildren(createEmptyState(t("initialEmpty")));
      resultsSummary.textContent = t("initialResultsSummary");
      resultCount.textContent = t("ready");
  }
}

function renderRecommendationResponse(body) {
  const recommendations = Array.isArray(body.recommendations) ? body.recommendations : [];
  resultCount.textContent = t("resultCount", { count: recommendations.length });
  availabilityNotice.textContent = currentLocale === "ja" && body.availabilityNote
    ? body.availabilityNote
    : t("availabilityNotice");
  availabilityNotice.hidden = false;
  travelNotice.textContent = currentLocale === "ja" && body.travelEstimateNote
    ? body.travelEstimateNote
    : localizedTravelNotice(recommendations);
  travelNotice.hidden = false;

  if (recommendations.length === 0) {
    results.replaceChildren(createEmptyState(t("emptyResults"), true));
    resultsSummary.textContent = t("emptyResultsSummary");
    return;
  }

  const rendered = [createPrimaryRecommendation(recommendations[0])];
  if (recommendations.length > 1) {
    rendered.push(createAlternativeRecommendations(recommendations.slice(1)));
  }
  results.replaceChildren(...rendered);
  resultsSummary.textContent = t("successResultsSummary");
}

function localizedTravelNotice(recommendations) {
  const kinds = recommendations.map((recommendation) => recommendation?.travelEstimateKind);
  if (kinds.length === 0 || kinds.some((kind) => kind !== "google_routes" && kind !== "straight_line")) {
    return t("travelNotice");
  }
  const uniqueKinds = new Set(kinds);
  if (uniqueKinds.size > 1) {
    return t("travelNoticeMixed");
  }
  return t(uniqueKinds.has("google_routes") ? "travelNoticeGoogle" : "travelNoticeStraight");
}

function travelKindLabel(kind) {
  if (kind === "google_routes") {
    return t("travelKindGoogle");
  }
  if (kind === "straight_line") {
    return t("travelKindStraight");
  }
  return "";
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
    action.textContent = t("changeConditionsAction");
    action.addEventListener("click", () => {
      conditionDetails.open = true;
      conditionDetails.querySelector("summary").focus();
    });
    container.append(action);
  }
  return container;
}

function createPrimaryRecommendation(recommendation) {
  const facility = recommendation.facility || {};
  const localizedFacility = localizeFacility(facility);
  const card = document.createElement("article");
  card.className = "result-card primary-result";

  const pickLabel = document.createElement("p");
  pickLabel.className = "pick-label";
  pickLabel.textContent = t("primaryPick");

  const title = document.createElement("h3");
  title.className = "result-title";
  title.textContent = localizedFacility.name;
  const address = document.createElement("p");
  address.className = "result-address";
  address.textContent = localizedFacility.address;

  const timing = document.createElement("div");
  timing.className = "session-timing";
  timing.append(
    createTimingMetric(t("arrivalEstimate"), formatClock(recommendation.arrivalAt)),
    createTimingMetric(t("skateEstimate"), formatMinutes(recommendation.estimatedSkateMinutes)),
    createTimingMetric(t("sessionEndEstimate"), formatClock(recommendation.sessionEndsAt)),
  );

  const meta = document.createElement("div");
  meta.className = "result-meta";
  meta.append(createTag(t("oneWayTravel", { minutes: recommendation.estimatedTravelMinutes }), "travel"));
  const estimateKindLabel = travelKindLabel(recommendation.travelEstimateKind);
  if (estimateKindLabel) {
    meta.append(createTag(estimateKindLabel, "travel-kind"));
  }
  for (const feature of (facility.features || []).slice(0, 4)) {
    meta.append(createTag(featureLabels[currentLocale][feature] || feature));
  }

  const reasons = document.createElement("ul");
  reasons.className = "reasons";
  for (const reason of selectRecommendationReasons(recommendation, 4)) {
    const item = document.createElement("li");
    item.textContent = localizeRecommendationReason(reason, recommendation);
    reasons.append(item);
  }

  const details = document.createElement("div");
  details.className = "facility-details";
  details.append(
    createDetail(t("price"), localizedFacility.price || t("needsConfirmation")),
    createDetail(t("reservation"), localizedFacility.reservation || t("needsConfirmation")),
    createDetail(t("closingTime"), formatClock(recommendation.facilityClosesAt)),
    createDetail(t("verifiedAt"), formatVerifiedAt(facility.verifiedAt)),
    createDetail(t("accessNotes"), localizedFacility.accessNotes || t("needsConfirmation")),
  );

  const scheduleNotes = createNotice(t("scheduleCaution"), localizedFacility.scheduleNotes);
  const availabilityNotice = createAvailabilityNotice(facility, localizedFacility);
  const rules = createNotice(t("rulesCaution"), localizedFacility.rules);
  const fallbackNotice = createTranslationFallback(localizedFacility.usesJapaneseFallback);
  const actions = createFacilityActions(facility, true);
  card.append(pickLabel, title, address, timing, meta, reasons, details);
  if (fallbackNotice) {
    card.append(fallbackNotice);
  }
  if (scheduleNotes) {
    card.append(scheduleNotes);
  }
  if (availabilityNotice) {
    card.append(availabilityNotice);
  }
  if (rules) {
    card.append(rules);
  }
  const media = createFacilityMedia(facility);
  if (media) {
    card.append(media);
  }
  card.append(actions);
  return card;
}

function createAvailabilityNotice(facility, localizedFacility) {
  const status = facility?.generalUseStatus;
  if (status !== "limited" && status !== "schedule_check_required" && !localizedFacility?.availabilityNote) {
    return null;
  }
  const note = localizedFacility?.availabilityNote || t("availabilityNotice");
  return createNotice(t("availabilityCaution"), [note]);
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

function createNotice(titleText, items) {
  if (!Array.isArray(items) || items.length === 0) {
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

function createTranslationFallback(usesJapaneseFallback) {
  if (currentLocale !== "en" || !usesJapaneseFallback) {
    return null;
  }
  const notice = document.createElement("p");
  notice.className = "translation-note";
  notice.textContent = t("translationFallback");
  return notice;
}

function createFacilityActions(facility, isPrimary) {
  const actions = document.createElement("div");
  actions.className = `card-actions${isPrimary ? "" : " alternative-actions"}`;

  const navigationLink = createExternalLink(
    t(isPrimary ? "goWithPlan" : "routeLink"),
    isPrimary ? "navigation-link" : "alternative-link",
    buildNavigationURL(facility.location),
    "navigation_opened",
    "route",
  );
  const sourceLink = createExternalLink(
    t("officialSource"),
    "source-link",
    facility.sourceUrl,
    "source_opened",
    "external-link",
  );
  const reportButton = document.createElement("button");
  reportButton.className = "report-button icon-text-action";
  reportButton.type = "button";
  appendIconText(reportButton, "file-pen", t("reportCorrection"));
  reportButton.addEventListener("click", () => openCorrectionDialog(facility, reportButton));
  actions.append(navigationLink, sourceLink, ...createSocialProfileLinks(facility), reportButton);
  return actions;
}

function createFacilityMedia(facility, isCompact = false) {
  const video = normalizedYouTubeVideo(facility?.media?.youtube);
  if (!video) {
    return null;
  }

  const section = document.createElement("section");
  section.className = `facility-media${isCompact ? " is-compact" : ""}`;
  section.setAttribute("aria-label", t("mediaSection"));

  const heading = document.createElement("h4");
  heading.textContent = t("youtubeVideo");
  const title = document.createElement("p");
  title.className = "facility-media-title";
  title.textContent = video.title;
  const privacy = document.createElement("p");
  privacy.className = "facility-media-privacy";
  privacy.textContent = t("youtubePrivacyNotice");

  const controls = document.createElement("div");
  controls.className = "facility-media-actions";
  const showVideo = document.createElement("button");
  showVideo.className = "video-button icon-text-action";
  showVideo.type = "button";
  appendIconText(showVideo, "video", t("showVideo"));

  const externalLink = createExternalLink(
    t("openYouTube"),
    "youtube-link",
    buildYouTubeWatchURL(video.videoId),
    "video_external_opened",
    "external-link",
  );
  const player = document.createElement("div");
  player.className = "youtube-player";
  player.hidden = true;

  showVideo.addEventListener("click", () => {
    if (!player.hidden) {
      return;
    }
    recordProductEvent("video_embed_requested");
    const iframe = document.createElement("iframe");
    iframe.src = buildYouTubeEmbedURL(video.videoId);
    iframe.title = video.title;
    iframe.loading = "lazy";
    iframe.referrerPolicy = "strict-origin-when-cross-origin";
    iframe.allow = "accelerometer; encrypted-media; gyroscope; picture-in-picture";
    iframe.allowFullscreen = true;
    iframe.addEventListener("load", () => recordProductEvent("video_embed_loaded"), { once: true });
    player.replaceChildren(iframe);
    player.hidden = false;
    showVideo.disabled = true;
  });

  controls.append(showVideo, externalLink);
  section.append(heading, title, privacy, controls, player);
  return section;
}

function normalizedYouTubeVideo(video) {
  if (video?.provider !== "youtube") {
    return null;
  }

  const videoId = typeof video?.videoId === "string" ? video.videoId.trim() : "";
  if (!/^[A-Za-z0-9_-]{11}$/.test(videoId)) {
    return null;
  }
  const title = typeof video.title === "string" && video.title.trim()
    ? video.title.trim()
    : t("youtubeVideo");
  return { videoId, title };
}

function buildYouTubeEmbedURL(videoId) {
  return `https://www.youtube-nocookie.com/embed/${encodeURIComponent(videoId)}?autoplay=0`;
}

function buildYouTubeWatchURL(videoId) {
  return `https://www.youtube.com/watch?v=${encodeURIComponent(videoId)}`;
}

function createSocialProfileLinks(facility) {
  const links = [];
  const seenPlatforms = new Set();
  for (const socialLink of Array.isArray(facility?.socialLinks) ? facility.socialLinks : []) {
    const platform = typeof socialLink?.platform === "string" ? socialLink.platform.toLowerCase() : "";
    const href = normalizedSocialProfileURL(platform, socialLink?.url);
    if (!href || seenPlatforms.has(platform)) {
      continue;
    }
    seenPlatforms.add(platform);
    links.push(createExternalLink(
      platform === "instagram" ? t("openInstagram") : t("openX"),
      `social-link social-link-${platform}`,
      href,
      "social_profile_opened",
      platform,
    ));
  }
  return links;
}

function normalizedSocialProfileURL(platform, value) {
  const allowedHosts = {
    instagram: new Set(["instagram.com", "www.instagram.com"]),
    x: new Set(["x.com", "www.x.com"]),
  };
  if (!allowedHosts[platform] || typeof value !== "string") {
    return "";
  }
  try {
    const url = new URL(value);
    if (url.protocol !== "https:" || !allowedHosts[platform].has(url.hostname.toLowerCase())) {
      return "";
    }
    return url.href;
  } catch {
    return "";
  }
}

function createAlternativeRecommendations(recommendations) {
  const alternatives = document.createElement("details");
  alternatives.className = "alternatives";
  const summary = document.createElement("summary");
  summary.textContent = t("alternatives", { count: recommendations.length });
  const list = document.createElement("div");
  list.className = "alternative-list";

  for (const recommendation of recommendations) {
    const facility = recommendation.facility || {};
    const localizedFacility = localizeFacility(facility);
    const row = document.createElement("article");
    row.className = "alternative-row";
    const copy = document.createElement("div");
    copy.className = "alternative-copy";
    const title = document.createElement("h3");
    title.textContent = localizedFacility.name;
    const timing = document.createElement("p");
    timing.className = "alternative-timing";
    timing.textContent = t("alternativeTiming", {
      travel: recommendation.estimatedTravelMinutes,
      skate: formatMinutes(recommendation.estimatedSkateMinutes),
    });
    const estimateKind = travelKindLabel(recommendation.travelEstimateKind);
    if (estimateKind) {
      const estimateKindText = document.createElement("span");
      estimateKindText.className = "alternative-travel-kind";
      estimateKindText.textContent = estimateKind;
      timing.append(" · ", estimateKindText);
    }
    const reason = document.createElement("p");
    reason.className = "alternative-reason";
    const reasonLabel = document.createElement("strong");
    reasonLabel.textContent = `${t("alternativeReasonLabel")}: `;
    reason.append(reasonLabel, localizeRecommendationReason(selectRecommendationReasons(recommendation, 1)[0], recommendation));
    const verified = document.createElement("p");
    verified.className = "alternative-verified";
    verified.textContent = `${t("verifiedAt")}: ${formatVerifiedAt(facility.verifiedAt)}`;
    copy.append(title, timing, reason, verified);
    const fallbackNotice = createTranslationFallback(localizedFacility.usesJapaneseFallback);
    if (fallbackNotice) {
      copy.append(fallbackNotice);
    }
    const media = createFacilityMedia(facility, true);
    if (media) {
      copy.append(media);
    }
    row.append(copy, createFacilityActions(facility, false));
    list.append(row);
  }

  alternatives.append(summary, list);
  return alternatives;
}

function selectRecommendationReasons(recommendation, limit) {
  const allReasons = Array.isArray(recommendation.reasons) ? recommendation.reasons : [];
  const decisionReasons = allReasons.filter(
    (reason) => reason?.code !== "travel_estimate" && reason?.code !== "session_time_estimate",
  );
  const selected = decisionReasons.length > 0 ? decisionReasons : allReasons;
  return selected.length > 0 ? selected.slice(0, limit) : [{ code: "fallback" }];
}

function localizeRecommendationReason(reason, recommendation) {
  switch (reason?.code) {
    case "purpose_match": {
      const purposeKeys = {
        basics: "reasonPurposeBasics",
        street: "reasonPurposeStreet",
        transition: "reasonPurposeTransition",
      };
      return t(purposeKeys[lastSearchContext?.purpose] || "reasonFallback");
    }
    case "beginner_friendly":
      return t("reasonBeginner");
    case "mood_match": {
      const moodKeys = {
        focused: "reasonMoodFocused",
        easygoing: "reasonMoodEasygoing",
        challenge: "reasonMoodChallenge",
      };
      return t(moodKeys[lastSearchContext?.mood] || "reasonFallback");
    }
    case "session_time_estimate":
      return t("reasonSessionTime", { minutes: recommendation.estimatedSkateMinutes });
    case "travel_estimate":
      return t("reasonTravel", { minutes: recommendation.estimatedTravelMinutes });
    default:
      return currentLocale === "ja" && typeof reason?.message === "string" && reason.message.trim()
        ? reason.message
        : t("reasonFallback");
  }
}

function localizeFacility(facility) {
  const englishTranslation = facility?.englishTranslation;
  const hasEnglishTranslation = isCompleteEnglishTranslation(englishTranslation);
  const translation = currentLocale === "en" && hasEnglishTranslation ? englishTranslation : null;
  return {
    name: translatedString(translation?.name, facility.name),
    address: translatedString(translation?.address, facility.address),
    price: translatedString(translation?.price, facility.price),
    reservation: translatedString(translation?.reservation, facility.reservation),
    scheduleNotes: translatedList(translation?.scheduleNotes, facility.scheduleNotes),
    availabilityNote: translatedString(translation?.availabilityNote, facility.availabilityNote),
    rules: translatedList(translation?.rules, facility.rules),
    accessNotes: translatedString(translation?.accessNotes, facility.access?.notes),
    usesJapaneseFallback: currentLocale === "en" && !hasEnglishTranslation,
  };
}

function isCompleteEnglishTranslation(translation) {
  if (!translation || typeof translation !== "object") {
    return false;
  }
  const requiredStrings = [
    translation.name,
    translation.address,
    translation.price,
    translation.reservation,
    translation.accessNotes,
  ];
  return requiredStrings.every((value) => typeof value === "string" && value.trim())
    && Array.isArray(translation.scheduleNotes)
    && Array.isArray(translation.rules);
}

function translatedString(translatedValue, fallbackValue) {
  if (typeof translatedValue === "string" && translatedValue.trim()) {
    return translatedValue;
  }
  return typeof fallbackValue === "string" ? fallbackValue : "";
}

function translatedList(translatedValues, fallbackValues) {
  if (Array.isArray(translatedValues)) {
    return translatedValues.filter((value) => typeof value === "string" && value.trim());
  }
  return Array.isArray(fallbackValues)
    ? fallbackValues.filter((value) => typeof value === "string" && value.trim())
    : [];
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
    return t("needsConfirmation");
  }
  return new Intl.DateTimeFormat(currentLocale === "ja" ? "ja-JP" : "en-GB", {
    year: "numeric",
    month: "short",
    day: "2-digit",
    timeZone: "Asia/Tokyo",
  }).format(date);
}

function formatClock(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return t("needsConfirmation");
  }
  return new Intl.DateTimeFormat(currentLocale === "ja" ? "ja-JP" : "en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "Asia/Tokyo",
  }).format(date);
}

function formatMinutes(value) {
  const minutes = Number(value);
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return t("needsConfirmation");
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (currentLocale === "en") {
    if (hours === 0) {
      return `${remainingMinutes} min`;
    }
    return remainingMinutes === 0 ? `${hours} hr` : `${hours} hr ${remainingMinutes} min`;
  }
  if (hours === 0) {
    return `${remainingMinutes}分`;
  }
  return remainingMinutes === 0 ? `${hours}時間` : `${hours}時間${remainingMinutes}分`;
}

function buildNavigationURL(location) {
  const latitude = Number(location?.latitude);
  const longitude = Number(location?.longitude);
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
    return "https://www.google.com/maps";
  }
  const parameters = new URLSearchParams({
    api: "1",
    destination: `${latitude},${longitude}`,
  });
  const travelMode = {
    public_transit: "transit",
    car: "driving",
    bicycle: "bicycling",
    walk: "walking",
  }[lastSearchContext?.transport];
  if (travelMode) {
    parameters.set("travelmode", travelMode);
  }

  const origin = lastSearchContext?.origin;
  const originLatitude = Number(origin?.latitude);
  const originLongitude = Number(origin?.longitude);
  if (origin?.mode === "specified_location"
    && Number.isFinite(originLatitude)
    && Number.isFinite(originLongitude)) {
    parameters.set("origin", `${originLatitude},${originLongitude}`);
  }
  return `https://www.google.com/maps/dir/?${parameters.toString()}`;
}

function createExternalLink(label, className, href, productEvent = "", iconName = "") {
  const link = document.createElement("a");
  link.className = `${className}${iconName ? " icon-text-action" : ""}`;
  link.href = safeHTTPURL(href);
  link.target = "_blank";
  link.rel = "noopener noreferrer";
  link.setAttribute("aria-label", label);
  if (iconName) {
    appendIconText(link, iconName, label);
  } else {
    link.textContent = label;
  }
  if (productEvent) {
    link.addEventListener("click", () => recordProductEvent(productEvent));
  }
  return link;
}

function appendIconText(element, iconName, label) {
  element.replaceChildren(createIcon(iconName));
  const text = document.createElement("span");
  text.className = "action-label";
  text.textContent = label;
  element.append(text);
}

function createIcon(name) {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.classList.add("action-icon");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("fill", "none");
  svg.setAttribute("stroke", "currentColor");
  svg.setAttribute("stroke-width", "2");
  svg.setAttribute("stroke-linecap", "round");
  svg.setAttribute("stroke-linejoin", "round");

  const shapes = {
    route: [
      ["circle", { cx: "6", cy: "19", r: "3" }],
      ["path", { d: "M9 19h6.5a3.5 3.5 0 0 0 0-7H9" }],
      ["path", { d: "m5 6 3-3 3 3" }],
      ["path", { d: "M8 3v9" }],
    ],
    "external-link": [
      ["path", { d: "M15 3h6v6" }],
      ["path", { d: "M10 14 21 3" }],
      ["path", { d: "M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" }],
    ],
    "file-pen": [
      ["path", { d: "M12 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" }],
      ["path", { d: "M18.4 2.6a2.1 2.1 0 0 1 3 3L12 15l-4 1 1-4Z" }],
    ],
    video: [
      ["path", { d: "m22 8-6 4 6 4V8Z" }],
      ["rect", { x: "2", y: "6", width: "14", height: "12", rx: "2" }],
    ],
    instagram: [
      ["rect", { x: "3", y: "3", width: "18", height: "18", rx: "5" }],
      ["path", { d: "M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37Z" }],
      ["path", { d: "M17.5 6.5h.01" }],
    ],
    x: [
      ["path", { d: "M5 4h3.5l3.8 5L16.4 4H19l-5.4 6.2L20 20h-3.5l-4.2-5.5L7.6 20H5l5.9-6.8Z" }],
    ],
  };
  for (const [tagName, attributes] of shapes[name] || []) {
    const shape = document.createElementNS("http://www.w3.org/2000/svg", tagName);
    for (const [attributeName, value] of Object.entries(attributes)) {
      shape.setAttribute(attributeName, value);
    }
    svg.append(shape);
  }
  return svg;
}

function safeHTTPURL(value) {
  try {
    const parsed = new URL(value, window.location.href);
    if (parsed.protocol === "http:" || parsed.protocol === "https:") {
      return parsed.href;
    }
  } catch {
    // Facility URLs are validated by the API; this is a defensive UI fallback.
  }
  return "#";
}

function openCorrectionDialog(facility, trigger) {
  correctionRequestVersion += 1;
  activeCorrectionFacility = facility;
  correctionTrigger = trigger;
  correctionForm.reset();
  correctionFacilityID.value = facility.facilityId || "";
  correctionSuccess.hidden = true;
  correctionForm.hidden = false;
  isCorrectionSubmitting = false;
  correctionState = { type: "idle", messageKey: "", reportId: "", nextAction: "" };
  clearCorrectionValidation();
  updateContactConsentState();
  updateCorrectionFacilityLabel();
  renderCorrectionState();

  if (typeof correctionDialog.showModal === "function") {
    correctionDialog.showModal();
  } else {
    correctionDialog.setAttribute("open", "");
  }
  correctionCategory.focus();
}

function closeCorrectionDialog() {
  correctionRequestVersion += 1;
  isCorrectionSubmitting = false;
  if (typeof correctionDialog.close === "function" && correctionDialog.open) {
    correctionDialog.close();
  } else {
    correctionDialog.removeAttribute("open");
  }
}

function updateCorrectionFacilityLabel() {
  if (!activeCorrectionFacility) {
    correctionFacility.textContent = "";
    return;
  }
  const localizedFacility = localizeFacility(activeCorrectionFacility);
  correctionFacility.textContent = `${localizedFacility.name} · ${activeCorrectionFacility.facilityId || ""}`;
}

function updateContactConsentState() {
  const hasContact = correctionContact.value.trim() !== "";
  correctionContactConsent.disabled = !hasContact || isCorrectionSubmitting;
  if (!hasContact) {
    correctionContactConsent.checked = false;
  }
}

async function submitCorrection() {
  if (isCorrectionSubmitting) {
    return;
  }
  const validation = validateCorrectionSubmission();
  if (!validation.isValid) {
    correctionState = { type: "validation", messageKey: validation.messageKey, reportId: "", nextAction: "" };
    renderCorrectionState();
    validation.field.focus();
    return;
  }

  const requestBody = {
    facilityId: correctionFacilityID.value,
    category: correctionCategory.value,
    details: correctionDetails.value.trim(),
    evidenceUrl: correctionEvidenceURL.value.trim(),
    contact: correctionContact.value.trim(),
    contactConsent: correctionContact.value.trim() !== "" && correctionContactConsent.checked,
  };

  isCorrectionSubmitting = true;
  const requestVersion = ++correctionRequestVersion;
  correctionState = { type: "submitting", messageKey: "correctionSendingStatus", reportId: "", nextAction: "" };
  renderCorrectionState();

  try {
    const response = await fetch("/api/corrections", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(requestBody),
    });
    const body = await parseJSON(response);
    if (requestVersion !== correctionRequestVersion) {
      return;
    }
    if (!response.ok) {
      throw new LocalizedError(response.status === 400 || response.status === 422
        ? "correctionServerValidation"
        : "correctionSubmitFailed");
    }
    if (!body || typeof body.reportId !== "string" || !body.reportId.trim()) {
      throw new LocalizedError("correctionSubmitFailed");
    }

    correctionState = {
      type: "success",
      messageKey: "",
      reportId: body.reportId,
      nextAction: typeof body.nextAction === "string" ? body.nextAction : "",
    };
  } catch (error) {
    if (requestVersion !== correctionRequestVersion) {
      return;
    }
    correctionState = {
      type: "error",
      messageKey: error instanceof LocalizedError ? error.messageKey : "correctionSubmitFailed",
      reportId: "",
      nextAction: "",
    };
  } finally {
    if (requestVersion !== correctionRequestVersion) {
      return;
    }
    isCorrectionSubmitting = false;
    renderCorrectionState();
  }
}

function validateCorrectionSubmission() {
  clearCorrectionValidation();
  if (!correctionCategories.has(correctionCategory.value)) {
    return invalidCorrection(correctionCategory, "validationCategory");
  }

  const details = correctionDetails.value.trim();
  if (!details) {
    return invalidCorrection(correctionDetails, "validationDetailsRequired");
  }
  if ([...details].length < 10 || [...details].length > 1000) {
    return invalidCorrection(correctionDetails, "validationDetailsLength");
  }

  const evidenceURL = correctionEvidenceURL.value.trim();
  if (evidenceURL && (!isValidHTTPSURL(evidenceURL) || evidenceURL.length > 500)) {
    return invalidCorrection(correctionEvidenceURL, "validationEvidenceUrl");
  }

  const contact = correctionContact.value.trim();
  if (contact && (!correctionContact.validity.valid || contact.length > 254)) {
    return invalidCorrection(correctionContact, "validationContact");
  }
  if (contact && !correctionContactConsent.checked) {
    return invalidCorrection(correctionContactConsent, "validationConsent");
  }

  return { isValid: true };
}

function invalidCorrection(field, messageKey) {
  field.setAttribute("aria-invalid", "true");
  return { isValid: false, field, messageKey };
}

function clearCorrectionValidation() {
  for (const field of correctionForm.querySelectorAll("[aria-invalid]")) {
    field.removeAttribute("aria-invalid");
  }
}

function isValidHTTPSURL(value) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === "https:" && Boolean(parsed.host);
  } catch {
    return false;
  }
}

function renderCorrectionState() {
  updateCorrectionFacilityLabel();
  if (!correctionForm || !correctionSuccess) {
    return;
  }

  const isSubmitting = correctionState.type === "submitting";
  const isSuccess = correctionState.type === "success";
  correctionForm.hidden = isSuccess;
  correctionSuccess.hidden = !isSuccess;
  correctionForm.setAttribute("aria-busy", String(isSubmitting));

  for (const field of [correctionCategory, correctionDetails, correctionEvidenceURL, correctionContact]) {
    field.disabled = isSubmitting;
  }
  correctionSubmitButton.disabled = isSubmitting;
  updateContactConsentState();

  if (isSuccess) {
    correctionReportID.textContent = correctionState.reportId;
    correctionNextAction.textContent = currentLocale === "ja" && correctionState.nextAction
      ? correctionState.nextAction
      : t("correctionNextAction");
    correctionStatus.textContent = "";
    correctionSubmitButton.textContent = t("submitCorrection");
    return;
  }

  correctionStatus.classList.toggle("is-error", correctionState.type === "error" || correctionState.type === "validation");
  correctionStatus.textContent = correctionState.messageKey ? t(correctionState.messageKey) : "";
  if (isSubmitting) {
    correctionSubmitButton.textContent = t("submittingCorrection");
  } else if (correctionState.type === "error") {
    correctionSubmitButton.textContent = t("retryCorrection");
  } else {
    correctionSubmitButton.textContent = t("submitCorrection");
  }
}

function setFormStatus(messageKey) {
  formStatusKey = messageKey;
  renderFormStatus();
}

function renderFormStatus() {
  formStatus.textContent = formStatusKey ? t(formStatusKey) : "";
}

function recordInputStarted() {
  if (hasRecordedInputStarted) {
    return;
  }
  hasRecordedInputStarted = true;
  recordProductEvent("input_started");
}

function recordProductEvent(eventName) {
  if (!allowedProductEvents.has(eventName)) {
    return;
  }
  void fetch("/api/events", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ event: eventName }),
    keepalive: true,
  }).catch(() => {
    // Product metrics are best-effort and must never block the user's flow.
  });
}

async function parseJSON(response) {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

function prefersReducedMotion() {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
