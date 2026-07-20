import {
  expect,
  test,
  type APIRequestContext,
  type Page,
} from "@playwright/test";

type Facility = {
  facilityId: string;
  name: string;
  sourceUrl: string;
  verifiedAt: string;
  location: {
    latitude: number;
    longitude: number;
  };
};

type RecommendationReason = {
  code: string;
  message: string;
};

type Recommendation = {
  facility: Facility;
  reasons: RecommendationReason[];
  distanceKm: number;
  estimatedTravelMinutes: number;
  estimatedSkateMinutes: number;
  arrivalAt: string;
  facilityClosesAt: string;
  sessionEndsAt: string;
  travelEstimateKind: string;
};

type RecommendationResponse = {
  recommendations: Recommendation[];
  travelEstimateNote: string;
  availabilityNote: string;
};

type NavigationExpectation = {
  origin?: string | null;
  travelMode?: "transit" | "driving" | "bicycling" | "walking";
};

async function loadFacilities(request: APIRequestContext): Promise<Facility[]> {
  const response = await request.get("/api/facilities?activity=skateboard");
  expect(response.ok()).toBe(true);

  const body = await response.json() as { facilities?: Facility[] };
  expect(Array.isArray(body.facilities)).toBe(true);
  return body.facilities ?? [];
}

function buildRecommendationResponse(facilities: Facility[]): RecommendationResponse {
  const recommendations = facilities.slice(0, 3).map((facility, index) => {
    const arrivalAt = new Date(Date.UTC(2026, 6, 20, 3, 20 + index * 10));
    const sessionEndsAt = new Date(arrivalAt.getTime() + 90 * 60_000);
    const facilityClosesAt = new Date(Date.UTC(2026, 6, 20, 13, 0));

    return {
      facility,
      reasons: [{
        code: `e2e_reason_${index + 1}`,
        message: `E2E候補${index + 1}の推薦根拠です。`,
      }],
      distanceKm: index + 1.2,
      estimatedTravelMinutes: 10 + index * 5,
      estimatedSkateMinutes: 90,
      arrivalAt: arrivalAt.toISOString(),
      facilityClosesAt: facilityClosesAt.toISOString(),
      sessionEndsAt: sessionEndsAt.toISOString(),
      travelEstimateKind: "straight_line",
    };
  });

  return {
    recommendations,
    travelEstimateNote: "E2E用の移動時間注記です。",
    availabilityNote: "E2E用の営業時間注記です。",
  };
}

async function mockRecommendationResponse(
  page: Page,
  response: RecommendationResponse,
): Promise<void> {
  await page.route("**/api/recommendations", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(response),
    });
  });
}

async function openConditionDetails(page: Page): Promise<void> {
  await page.getByText("条件を変更", { exact: true }).click();
}

async function assertRecommendationCards(
  page: Page,
  recommendations: Recommendation[],
  navigationExpectation: NavigationExpectation = {},
): Promise<void> {
  if (recommendations.length > 1) {
    const otherOptions = page.getByText(
      new RegExp(`^ほかの候補 ${recommendations.length - 1}件$`),
    );
    await expect(otherOptions).toBeVisible();
    await otherOptions.click();
  }

  const cards = page.getByRole("article");
  await expect(cards).toHaveCount(recommendations.length);
  for (const recommendation of recommendations) {
    const facility = recommendation.facility;
    const heading = page.getByRole("heading", {
      name: facility.name,
      exact: true,
    });
    const card = cards.filter({ has: heading });
    await expect(card).toHaveCount(1);
    await expect(card).toBeVisible();

    const decisionReasons = recommendation.reasons.filter(
      (reason) => reason.code !== "travel_estimate"
        && reason.code !== "session_time_estimate",
    );
    const displayedReason = decisionReasons[0] ?? recommendation.reasons[0];
    if (displayedReason) {
      await expect(card).toContainText(displayedReason.message);
    }
    await expect(card.getByText(/情報確認日/)).toBeVisible();

    const sourceLink = card.getByRole("link", {
      name: "公式情報",
      exact: true,
    });
    await expect(sourceLink).toHaveAttribute("href", facility.sourceUrl);

    const navigationLink = card.getByRole("link", {
      name: /^(このプランで行く|経路を見る)$/,
    });
    const navigationHref = await navigationLink.getAttribute("href");
    expect(navigationHref).not.toBeNull();
    const navigationURL = new URL(navigationHref as string);
    expect(navigationURL.origin).toBe("https://www.google.com");
    expect(navigationURL.pathname).toBe("/maps/dir/");
    expect(navigationURL.searchParams.get("destination")).toBe(
      `${facility.location.latitude},${facility.location.longitude}`,
    );
    if (navigationExpectation.origin !== undefined) {
      expect(navigationURL.searchParams.get("origin")).toBe(navigationExpectation.origin);
    }
    if (navigationExpectation.travelMode !== undefined) {
      expect(navigationURL.searchParams.get("travelmode")).toBe(
        navigationExpectation.travelMode,
      );
    }
  }
}

test("日本語と英語を切り替えて選択を保持できる", async ({ page }) => {
  await page.goto("/");

  await expect(page.locator("html")).toHaveAttribute("lang", "ja");
  await expect(page.getByRole("heading", {
    name: "今から滑る場所を決める",
  })).toBeVisible();

  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page).toHaveTitle("spot-diggz | Today's skate spot");
  await expect(page.getByRole("heading", {
    name: "Choose where to skate now",
  })).toBeVisible();

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await expect(page.getByRole("heading", {
    name: "Today's recommendations",
  })).toBeVisible();

  await page.getByRole("button", { name: "日本語", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("lang", "ja");
  await expect(page.getByRole("heading", {
    name: "今から滑る場所を決める",
  })).toBeVisible();

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("lang", "ja");
});

test("代表地点からGo APIの推薦結果を表示する", async ({ page }) => {
  await page.goto("/");
  await openConditionDetails(page);
  await page.getByLabel("出発地点").selectOption("namba-station");
  await page.getByLabel("レベル").selectOption("returning");
  await page.getByLabel("使える時間").selectOption("240");

  const recommendationRequestPromise = page.waitForRequest((request) =>
    request.method() === "POST"
      && new URL(request.url()).pathname === "/api/recommendations"
  );
  const recommendationResponsePromise = page.waitForResponse((response) =>
    response.request().method() === "POST"
      && new URL(response.url()).pathname === "/api/recommendations"
  );
  await page.getByRole("button", {
    name: "この条件でおすすめを更新",
  }).click();

  const [request, response] = await Promise.all([
    recommendationRequestPromise,
    recommendationResponsePromise,
  ]);
  expect(response.ok()).toBe(true);
  const input = request.postDataJSON() as {
    origin?: { mode?: string; latitude?: number; longitude?: number };
  };
  expect(input.origin?.mode).toBe("specified_location");
  expect(typeof input.origin?.latitude).toBe("number");
  expect(typeof input.origin?.longitude).toBe("number");

  const body = await response.json() as RecommendationResponse;
  expect(body.recommendations.length).toBeGreaterThan(0);
  await expect(page.getByText(
    "通常営業時間内の候補をおすすめ順に選びました",
    { exact: true },
  )).toBeVisible();
  await assertRecommendationCards(page, body.recommendations, {
    origin: "34.6662,135.5001",
    travelMode: "transit",
  });
});

test("全候補に推薦根拠、公式出典、外部ナビを表示する", async ({
  page,
  request,
}) => {
  const facilities = await loadFacilities(request);
  expect(facilities.length).toBeGreaterThanOrEqual(3);
  const mockedResponse = buildRecommendationResponse(facilities);
  await mockRecommendationResponse(page, mockedResponse);

  await page.goto("/");
  await page.getByRole("button", {
    name: "おまかせで今から滑る",
  }).click();

  await assertRecommendationCards(page, mockedResponse.recommendations, {
    origin: "34.7025,135.496",
    travelMode: "transit",
  });
});

test("施設単位の訂正dialogで報告を受け付ける", async ({
  page,
  request,
}) => {
  const facilities = await loadFacilities(request);
  const mockedResponse = buildRecommendationResponse(facilities);
  expect(mockedResponse.recommendations.length).toBeGreaterThan(0);
  await mockRecommendationResponse(page, mockedResponse);

  await page.goto("/");
  await page.getByRole("button", {
    name: "おまかせで今から滑る",
  }).click();

  const selectedFacility = mockedResponse.recommendations[0].facility;
  const selectedCard = page.getByRole("article").filter({
    has: page.getByRole("heading", {
      name: selectedFacility.name,
      exact: true,
    }),
  });
  await selectedCard.getByRole("button", {
    name: "情報の誤りを報告",
  }).click();

  const dialog = page.getByRole("dialog", {
    name: "情報の誤りを報告",
  });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("combobox", {
    name: /訂正する項目/,
  }).selectOption("hours");
  await dialog.getByRole("textbox", {
    name: /訂正内容/,
  }).fill("E2Eで営業時間の訂正受付を確認しています。");
  await dialog.getByRole("textbox", {
    name: /根拠URL/,
  }).fill("https://example.com/official-correction");

  const correctionRequestPromise = page.waitForRequest((request) =>
    request.method() === "POST"
      && new URL(request.url()).pathname === "/api/corrections"
  );
  const correctionResponsePromise = page.waitForResponse((response) =>
    response.request().method() === "POST"
      && new URL(response.url()).pathname === "/api/corrections"
  );
  await dialog.getByRole("button", { name: "報告を送信" }).click();

  const [correctionRequest, correctionResponse] = await Promise.all([
    correctionRequestPromise,
    correctionResponsePromise,
  ]);
  expect(correctionResponse.status()).toBe(202);
  expect(correctionRequest.postDataJSON()).toMatchObject({
    facilityId: selectedFacility.facilityId,
    category: "hours",
  });
  const receipt = await correctionResponse.json() as { reportId?: string };
  expect(receipt.reportId).toMatch(/^COR-[0-9a-f]{32}$/);
  await expect(dialog.getByText(
    "報告を受け付けました",
    { exact: true },
  )).toBeVisible();
  await expect(dialog.locator("code")).toHaveText(receipt.reportId as string);
});

test("現在地を拒否したとき代表地点への切替を案内する", async ({ page }) => {
  await page.addInitScript(() => {
    const permissionDeniedError = {
      code: 1,
      message: "Permission denied",
      PERMISSION_DENIED: 1,
      POSITION_UNAVAILABLE: 2,
      TIMEOUT: 3,
    } as GeolocationPositionError;
    Object.defineProperty(window.navigator, "geolocation", {
      configurable: true,
      value: {
        getCurrentPosition: (
          _success: PositionCallback,
          error?: PositionErrorCallback,
        ) => error?.(permissionDeniedError),
      },
    });
  });

  let recommendationRequestCount = 0;
  page.on("request", (request) => {
    if (request.method() === "POST"
      && new URL(request.url()).pathname === "/api/recommendations") {
      recommendationRequestCount += 1;
    }
  });

  await page.goto("/");
  await openConditionDetails(page);
  const currentLocationRadio = page.getByRole("radio", {
    name: "現在地",
    exact: true,
  });
  await page.getByText("現在地", { exact: true }).click();
  await expect(currentLocationRadio).toBeChecked();
  await page.getByRole("button", { name: "現在地を確認" }).click();

  await expect(page.locator("#condition-details")).toHaveAttribute("open", "");
  await expect(page.getByRole("radio", {
    name: "地点を選ぶ",
    exact: true,
  })).toBeChecked();
  const departurePoint = page.getByLabel("出発地点");
  await expect(departurePoint).toBeVisible();
  await expect(departurePoint).toBeFocused();
  await expect(page.locator("#form-status")).toHaveText(
    "現在地を取得できませんでした。地点を選ぶか、位置情報を許可してください。",
  );
  await departurePoint.selectOption("namba-station");
  expect(recommendationRequestCount).toBe(0);
});

test("住所候補を選ばずに推薦すると選択を要求する", async ({ page }) => {
  const searchedLocation = {
    label: "テスト北駅（住所検索）",
    location: { latitude: 34.7001, longitude: 135.5001 },
  };
  await page.route("**/api/locations/search", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ results: [searchedLocation] }),
    });
  });

  let recommendationRequestCount = 0;
  page.on("request", (request) => {
    if (request.method() === "POST"
      && new URL(request.url()).pathname === "/api/recommendations") {
      recommendationRequestCount += 1;
    }
  });

  await page.goto("/");
  await openConditionDetails(page);
  await page.getByRole("searchbox", {
    name: "任意の駅名・住所",
  }).fill("大阪府テスト市北区1-1");
  await page.getByRole("button", { name: "検索", exact: true }).click();

  const candidate = page.getByRole("radio", { name: searchedLocation.label });
  await expect(candidate).toBeVisible();
  await page.getByRole("button", {
    name: "この条件でおすすめを更新",
  }).click();

  await expect(page.locator("#condition-details")).toHaveAttribute("open", "");
  await expect(candidate).toBeFocused();
  await expect(page.locator("#form-status")).toHaveText(
    "検索候補から出発地点を1件選んでください。",
  );
  expect(recommendationRequestCount).toBe(0);

  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(page.locator("#form-status")).toHaveText(
    "Select one departure point from the search results.",
  );
});

test("選択済み住所を編集した後は候補の再選択を要求する", async ({ page }) => {
  let recommendationCalls = 0;
  await page.route("**/api/locations/search", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        results: [{
          label: "兵庫県神戸市中央区",
          location: { latitude: 34.6901, longitude: 135.1955 },
        }],
      }),
    });
  });
  await page.route("**/api/recommendations", async (route) => {
    recommendationCalls += 1;
    await route.fulfill({ status: 500, body: "unexpected request" });
  });

  await page.goto("/");
  await openConditionDetails(page);
  const query = page.locator("#location-search-query");
  await query.fill("神戸駅");
  await page.getByRole("button", { name: "検索", exact: true }).click();
  await page.getByText("兵庫県神戸市中央区", { exact: true }).click();
  await query.fill("神戸駅 西口");
  await page.getByRole("button", { name: "この条件でおすすめを更新" }).click();

  await expect(page.locator("#form-status")).toHaveText(
    "検索候補から出発地点を1件選んでください。",
  );
  await expect(page.locator("#condition-details")).toHaveAttribute("open", "");
  await expect(query).toBeFocused();
  expect(recommendationCalls).toBe(0);
});

test("住所検索の候補を推薦の起点に使う", async ({ page }) => {
  const searchedLocation = {
    label: "テスト北駅（住所検索）",
    location: { latitude: 34.7001, longitude: 135.5001 },
  };
  let searchedQuery = "";
  await page.route("**/api/locations/search", async (route) => {
    expect(route.request().method()).toBe("POST");
    searchedQuery = (route.request().postDataJSON() as { query?: string }).query ?? "";
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ results: [searchedLocation] }),
    });
  });

  await page.goto("/");
  await openConditionDetails(page);
  await page.getByRole("searchbox", {
    name: "任意の駅名・住所",
  }).fill("大阪府テスト市北区1-1");
  await page.getByRole("button", { name: "検索", exact: true }).click();

  expect(searchedQuery).toBe("大阪府テスト市北区1-1");
  const searchedLocationRadio = page.getByRole("radio", {
    name: searchedLocation.label,
  });
  await expect(searchedLocationRadio).toBeVisible();
  await page.getByText(searchedLocation.label, { exact: true }).click();
  await expect(searchedLocationRadio).toBeChecked();
  await expect(page.getByText(
    `出発地点に「${searchedLocation.label}」を選びました。`,
    { exact: true },
  )).toBeVisible();

  const recommendationRequestPromise = page.waitForRequest((request) =>
    request.method() === "POST"
      && new URL(request.url()).pathname === "/api/recommendations"
  );
  const recommendationResponsePromise = page.waitForResponse((response) =>
    response.request().method() === "POST"
      && new URL(response.url()).pathname === "/api/recommendations"
  );
  await page.getByRole("button", {
    name: "この条件でおすすめを更新",
  }).click();

  const [recommendationRequest, recommendationResponse] = await Promise.all([
    recommendationRequestPromise,
    recommendationResponsePromise,
  ]);
  const input = recommendationRequest.postDataJSON() as {
    origin?: { mode?: string; latitude?: number; longitude?: number };
  };
  expect(input.origin).toMatchObject({
    mode: "specified_location",
    latitude: searchedLocation.location.latitude,
    longitude: searchedLocation.location.longitude,
  });
  expect(recommendationResponse.ok()).toBe(true);
  const body = await recommendationResponse.json() as RecommendationResponse;
  expect(body.recommendations.length).toBeGreaterThan(0);
  await assertRecommendationCards(page, body.recommendations, {
    origin: `${searchedLocation.location.latitude},${searchedLocation.location.longitude}`,
    travelMode: "transit",
  });

  await openConditionDetails(page);
  await page.getByRole("searchbox", {
    name: "任意の駅名・住所",
  }).fill("大阪府テスト市南区2-2");
  await expect(page.getByRole("radio", {
    name: searchedLocation.label,
  })).toHaveCount(0);
  await expect(page.locator("#location-search-status")).toBeEmpty();
});

test("Googleへの外部送信と端末保存の違いを操作前に日英で表示する", async ({ page }) => {
  await page.goto("/");
  await openConditionDetails(page);

  await expect(page.locator("#location-search-provider-notice")).toContainText(
    "住所文字列をGoogleへ送る場合があります",
  );
  await expect(page.locator("#location-search-provider-notice")).toContainText(
    "保存または永続化しません",
  );
  await page.getByText("現在地", { exact: true }).click();
  await expect(page.locator("#current-location-provider-notice")).toBeVisible();
  await expect(page.locator("#current-location-provider-notice")).toContainText(
    "現在地の座標をGoogleへ送る場合があります",
  );

  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(page.locator("#current-location-provider-notice")).toContainText(
    "separate from storage",
  );
  await page.getByText("Choose a station", { exact: true }).click();
  await expect(page.locator("#location-search-provider-notice")).toContainText(
    "not saved or persisted",
  );
});

test("外部ナビへpreset起点と4種類の交通手段を引き継ぐ", async ({
  page,
  request,
}) => {
  const facilities = await loadFacilities(request);
  const mockedResponse = buildRecommendationResponse(facilities.slice(0, 1));
  await mockRecommendationResponse(page, mockedResponse);
  const travelModes = [
    ["public_transit", "transit"],
    ["car", "driving"],
    ["bicycle", "bicycling"],
    ["walk", "walking"],
  ] as const;

  await page.goto("/");
  for (const [transport, googleTravelMode] of travelModes) {
    if (!(await page.locator("#condition-details").getAttribute("open"))) {
      await openConditionDetails(page);
    }
    await page.getByLabel("交通手段").selectOption(transport);
    const recommendationResponse = page.waitForResponse((response) =>
      response.request().method() === "POST"
        && new URL(response.url()).pathname === "/api/recommendations"
    );
    await page.getByRole("button", {
      name: "この条件でおすすめを更新",
    }).click();
    await recommendationResponse;

    const navigationHref = await page.locator(".navigation-link").getAttribute("href");
    expect(navigationHref).not.toBeNull();
    const navigationURL = new URL(navigationHref as string);
    expect(navigationURL.searchParams.get("origin")).toBe("34.7025,135.496");
    expect(navigationURL.searchParams.get("travelmode")).toBe(googleTravelMode);
  }
});

test("現在地からの外部ナビにはoriginを含めない", async ({ page, request }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window.navigator, "geolocation", {
      configurable: true,
      value: {
        getCurrentPosition: (success: PositionCallback) => success({
          coords: {
            latitude: 34.701,
            longitude: 135.501,
            accuracy: 10,
            altitude: null,
            altitudeAccuracy: null,
            heading: null,
            speed: null,
          },
          timestamp: Date.now(),
        } as GeolocationPosition),
      },
    });
  });
  const facilities = await loadFacilities(request);
  const mockedResponse = buildRecommendationResponse(facilities.slice(0, 1));
  await mockRecommendationResponse(page, mockedResponse);

  await page.goto("/");
  await openConditionDetails(page);
  await page.getByText("現在地", { exact: true }).click();
  await page.getByLabel("交通手段").selectOption("walk");
  await page.getByRole("button", {
    name: "この条件でおすすめを更新",
  }).click();

  const navigationLink = page.locator(".navigation-link");
  await expect(navigationLink).toBeVisible();
  const navigationHref = await navigationLink.getAttribute("href");
  expect(navigationHref).not.toBeNull();
  const navigationURL = new URL(navigationHref as string);
  expect(navigationURL.searchParams.get("origin")).toBeNull();
  expect(navigationURL.searchParams.get("travelmode")).toBe("walking");
});

test("0件時は実装済みの条件変更と営業時間内の再試行を案内する", async ({ page }) => {
  await mockRecommendationResponse(page, {
    recommendations: [],
    travelEstimateNote: "E2E用の移動時間注記です。",
    availabilityNote: "E2E用の営業時間注記です。",
  });
  await page.goto("/");
  await page.getByRole("button", {
    name: "おまかせで今から滑る",
  }).click();

  await expect(page.getByText(
    "今の条件で十分に滑れる候補がありません。出発地点・交通手段・利用可能時間を変えるか、営業時間内に再試行してください。",
    { exact: true },
  )).toBeVisible();
  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(page.getByText(
    "No spot leaves enough time to skate with these conditions. Change the departure point, transport, or available time, or try again during opening hours.",
    { exact: true },
  )).toBeVisible();
});
