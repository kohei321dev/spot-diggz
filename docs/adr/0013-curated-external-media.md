# ADR-0013 手動選定したYouTube動画と公式SNS導線を外部メディアとして扱う

- Status: Accepted
- Date: 2026-07-21
- Related: [Product Baseline](../product_baseline.md)
- Related: [ADR-0002](0002-facility-data-source-and-freshness.md)
- Related: [ADR-0008](0008-facility-catalog-api-and-storage.md)
- Related: [ADR-0009](0009-session-recommendation-ui.md)
- Related: [Security Baseline](../security/security-baseline.md)

## Context

MVPの推薦結果は施設の名称、条件、営業時間、公式情報、外部ナビを中心に表示する。行き先を選ぶ利用者には、施設のセクションや雰囲気を判断できる視覚情報も有用である。一方で、Google Maps、Apple Maps、SKEPA、SNS等の画像、投稿、フィードを無断で収集、複製、再配信すると、利用規約、著作権、肖像権、privacyの責任範囲が不明確になる。

施設情報の正確性を動画やSNSの投稿から推論してはならない。動画は古い、撮影場所が異なる、運用状況が変化している可能性があり、推薦条件、営業時間、ルール、初心者適性の根拠にはならない。また、任意のiframe HTMLや任意hostをcatalogから配信すると、XSS、clickjacking、意図しないthird-party通信、CSP形骸化の境界になる。

## Decision

1. 施設ごとに、運用者が手動で選定・確認したYouTube動画を任意で最大1件だけ表示できる。動画は施設の視覚的な参考情報であり、推薦score、鮮度、営業状態、ルールの判定には使わない。
2. catalogは任意のiframe HTML、任意embed URL、任意hostを保存しない。YouTube動画はprovider allowlist上の`youtube`、厳格に検証したvideo ID、元のwatch URL、選定・確認時刻、表示名、選定理由だけを構造化して保存する。UIは検証済みvideo IDと固定provider設定からembed URLを組み立てる。
3. 埋込先はYouTubeのprivacy-enhanced endpoint (`https://www.youtube-nocookie.com`)だけとし、`frame-src` CSPはこのoriginだけを許可する。親ページはYouTube playerの外側へoverlayを置かず、player内のYouTube表示、広告、操作、attributionを妨げない。
4. 動画を持つ推薦cardではYouTube iframeを初期表示する。`autoplay=0` とし、利用者の明示操作なしに再生しない。同じトグル操作でiframeを閉じて再表示できる。embedを表示する領域はYouTubeの最低表示サイズを満たす。
5. 動画を持つ推薦結果では、YouTubeへの通信と再生時の追加通信が起こり得ることをUIで明示する。`Referrer-Policy` は埋込playerがclient identityを受け取れる `strict-origin-when-cross-origin` とし、`noreferrer` を使わない。privacy-enhanced endpointでも、第三者送信がないとは表現しない。
6. iframeが利用できない、埋込が禁止されている、動画が削除・非公開になった、または利用者がiframeを閉じた場合、元のYouTube watch URLを新しいtabで開く外部リンクを表示する。外部リンクは `noopener` を付け、動画の閲覧は推薦・詳細表示を失敗させない。
7. SNSはInstagramとXだけを対象とし、運用者が公式性を確認したprofileへの外部リンクだけを許可する。catalogのSNS platform allowlistは`instagram`と`x`だけとし、HTTPS URLを構造化して保存する。UIはplatformを示すicon、テキスト名、外部リンクで表示する。YouTubeはSNS platformとして保存せず、第2項の動画metadataとwatch URLだけで扱う。投稿、ハッシュタグ、フィード、画像、動画、埋込、スクレイピング、OGP取得、SNS APIによる収集は行わない。
8. 動画とSNSの選定は、facility catalogの通常reviewで行う。運用者は施設との関係、公開性、表示名、リンク先、埋込可否、YouTube、Instagram、Xの利用規約・branding・privacy要件を確認し、確認日時を記録する。著作権、肖像権、privacy、規約上の問題を解消できない場合は掲載しない。
9. catalog validationは、動画が0件または1件であること、providerがallowlist内であること、video ID形式、watch URL host、SNS platformとhost、HTTPS、重複のないplatformを検証する。validation failureはcatalog startup failureとする。
10. 外部メディアを使うためにYouTube Data API、SNS API、credential、server-side fetch、proxy、asset保存を追加しない。動画とSNSはcatalogのread-only metadataおよびbrowserの明示的な外部遷移として扱う。

## Alternatives

### Google Maps、Apple Maps、SKEPA、SNS等から画像や投稿を収集する

視覚情報を増やせるが、利用規約、著作権、肖像権、削除要求、再配信、取得失敗への責任を持つ必要がある。MVPのcatalog品質運用を超えるため採用しない。

### InstagramやXの投稿・ハッシュタグを埋め込む

施設の最新情報へ誘導できる可能性があるが、投稿内容、追跡、表示崩れ、年齢制限、provider変更に依存する。SNSは公式profileへの外部リンクだけに限定する。

### 任意のvideo URLまたはiframe HTMLを運用者が登録する

provider追加が容易になるが、host validationとCSPが形骸化し、XSSとthird-party通信のreview境界を失う。固定provider設定とvideo IDからのURL組み立てを採用する。

### 初期表示でYouTubeサムネイルとiframeをloadしない

third-party通信は抑えられるが、利用者が動画の存在に気付かず、施設の雰囲気を比較する目的を果たしにくい。初期表示と、第三者通信の明示・自動再生禁止・開閉トグルを採用する。

### 動画を施設の正確性または推薦scoreへ使う

視覚的な情報量は増えるが、撮影時点・撮影場所・編集内容を検証できず、公式sourceに基づく推薦根拠を弱める。参考情報に限定する。

## Consequences

### Positive

- 推薦結果から、利用者が施設の雰囲気を補助的に確認できる
- 許可したprovider、URL形式、通信開始時点を明示できる
- 動画・SNSの障害や削除が推薦flowへ波及しない
- 画像・投稿の収集、asset管理、SNS feed運用を持たずに済む

### Negative

- 動画選定と規約確認を施設ごとに手動で続ける必要がある
- 動画がない施設とある施設で情報量に差が出る
- 動画を持つ推薦結果の初期表示時にYouTubeへ第三者通信が発生し、provider側の表示・広告・利用規約に依存する
- 動画の閲覧は施設の現況を保証しないため、公式情報と検証日時の確認を引き続き必要とする

## Verification

### Local verification

- catalog validatorが動画0件/1件、allowlist provider、video ID、watch URL、SNS HTTPS host、SNS platform重複を検証すること
- 任意iframe HTML、任意embed URL、未許可provider、複数動画をcatalogから受理しないこと
- 動画を持つ初期結果にYouTube iframeを1件作成し、`autoplay=0`、16:9表示、YouTube attributionと標準player controlを維持すること
- 動画の開閉トグルでiframeを閉じて再表示でき、再表示時に別のiframeを作成しないこと
- player load failureまたは埋込不可時に、YouTube外部リンクが利用でき、推薦結果が表示されたままであること
- Instagram/X linkがallowlistされたprofileだけを新しいtabで開くこと
- CSPの`frame-src`がYouTube privacy-enhanced originだけを許可し、Referrer-Policyが`strict-origin-when-cross-origin`であること
- keyboard、screen reader、mobileで動画・SNS操作のラベル、focus、44px以上のtap targetを確認すること
- log、metric、product eventにvideo ID、watch URL、SNS URL、title、利用者の再生履歴を記録しないこと

### External verification

- [未確認] 選定した動画ごとに、施設との関係、公開性、埋込可否、元URL、表示名を運用者が確認すること
- [未確認] production CSP、Referrer-Policy、YouTube player表示、外部リンクfallback、第三者通信のprivacy表示を実ブラウザで確認すること
- [未確認] YouTube、Instagram、Xの最新の利用規約、branding、privacy要件と、適用される著作権・肖像権・privacy要件をownerが確認すること

## Revisit Conditions

- 動画の選定・確認がcatalog運用上限を超える
- 外部メディアが推薦結果の理解や外部ナビ遷移を改善しない
- YouTubeまたはSNSの規約・privacy要件、CSP要件、埋込仕様が変わる
- 複数動画、画像、別provider、利用者投稿を扱う必要が生じ、その責任範囲とmoderatation方法を定義できる

## References

- [YouTube API Services Developer Policies](https://developers.google.com/youtube/terms/developer-policies)
- [YouTube embedded player requirements](https://developers.google.com/youtube/terms/required-minimum-functionality)
- [YouTube embedded players and player parameters](https://developers.google.com/youtube/player_parameters)
