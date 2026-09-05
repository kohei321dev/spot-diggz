# Web UI Specification

- Status: Current
- Related requirements: R-001–R-005、R-007、R-013–R-017
- Related decisions: ADR-0004、ADR-0009、ADR-0013、ADR-0014、ADR-0015

## Access

- ProductionのWeb UIと`/api/*`はGitHub OAuthで許可ownerだけが利用できます。
- 未認証でUIを開くとsign-in導線を表示し、未認証APIは401を返します。
- Local画面確認では`DEV_AUTH_BYPASS=1`を使用できますが、Productionでは常に無効です。

## Recommendation input

利用者は目的、気分、level、利用可能時間、検索位置、交通手段を指定します。既定条件によるワンクリック推薦を提供し、詳細条件は必要に応じて開きます。

検索位置は次から選びます。

- 公開された代表地点
- Browser Geolocationで得た現在位置
- Google Geocodingが有効な場合の駅名・住所検索

正確な座標と検索文字列はapplicationへ保存せず、responseへ再掲しません。Google有効時の外部送信境界は画面上で明示します。

## Recommendation result

- hard conditionを満たした候補を安定順序で最大3件表示します。
- 各candidateには順序、施設名、住所、到着目安、滑走目安、終了目安、移動目安、主要設備等の比較要点を配置します。
- おすすめ理由は検索条件に対する説明であり、施設の恒久属性として保存しません。
- 料金、予約、閉場目安、情報確認日、access補足をcandidateへ常時表示します。営業日の注意、一般利用の注意、利用ruleは施設詳細内へ配置します。
- 「ここに行く」と「公式情報を確認」の外部導線を常時利用できます。
- provider結果が概算の場合は、straight-line estimateであることと実経路確認が必要なことを表示します。

## Facility details

- 補助情報はcandidate card内の「施設の詳細を見る」を利用者が選んだ後に表示します。
- 詳細には利用rule、設備、access、情報源、検証時刻、訂正報告導線、登録済みmediaを含められます。
- 詳細を閉じてもcandidateの比較要点、訪問前確認、外部導線は利用できます。

## YouTube and social profiles

- YouTube動画は施設ごとに0または1件です。
- iframeは動画を含む施設詳細を開いた後だけ生成し、詳細を閉じている間は生成しません。
- `youtube-nocookie.com`の固定URL、`autoplay=0`を使い、任意URLをiframeへ渡しません。
- 同じtoggleで動画を閉じて再表示できます。埋込失敗時もcandidateを残し、通常のYouTube linkを利用できます。
- Instagram/Xは公式確認済みprofileへの外部linkだけを表示し、post、feed、hashtag、OGPを取得・表示しません。

## Facility list

- 独立した保存Lists UI、保存API、保存tableはありません。
- `GET /api/facilities`は認証済みownerへ施設catalogを返しますが、現在のWeb UIは保存済み施設一覧を提供しません。
- [`../research/ui/facility-list-sample.html`](../research/ui/facility-list-sample.html)は非正規の検討案であり、現行機能ではありません。

## Correction report

- 施設単位で訂正category、details、任意evidence URL、任意contactを送信できます。
- contactは明示同意がある場合だけ受理します。
- 正確な現在地、氏名、電話番号等の個人情報を入力しないよう案内します。
- 保存成功時はreceiptを表示し、catalogへ自動反映しません。

## Localization and accessibility

- 主要flow、施設の主要事実、errorを日本語・英語で表示します。
- iconだけで操作や状態を表さず、短いlabelとaccessible nameを持たせます。
- 主要controlはkeyboard操作でき、44 CSS px以上の操作領域を持ちます。
- 画面幅が変わっても横scrollへ依存せず、desktop/mobileの主要flowをE2Eで確認します。

## Error behavior

- 入力不正はfield単位で次の行動が分かる案内を表示します。
- 認証失敗はsign-inまたは再試行を案内します。
- Google Geocoding利用不能時は代表地点または現在地へ戻れるようにします。
- Google Routes失敗、media失敗、SNS未登録は推薦全体の失敗にしません。
- catalogが全件staleの場合、readinessは503とし、利用可能な推薦がないことを明示します。
