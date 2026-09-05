# ADR-0014 施設の補助情報とYouTube埋込を明示操作後に表示する

- Status: Accepted
- Date: 2026-07-22
- Type: Specification
- Related Issues: Incomplete — repository historyから対応Issueを特定できていない
- Related Pull Requests: [#304](https://github.com/kohei321dev/spot-diggz/pull/304)
- Affected Docs: `requirements.md`, `security.md`, `specifications/web-ui.md`
- Supersedes: [ADR-0013](0013-curated-external-media.md)（Decision 4と初期iframe loadのみ）
- Superseded By: None
- Related: [Product](../product.md)
- Related: [ADR-0009](0009-session-recommendation-ui.md)
- Related: [ADR-0013](0013-curated-external-media.md)
- Related: [Security](../security.md)

## Context

推薦cardは到着・滑走・終了時刻、推薦理由、料金、受付、注意事項、ルール、外部導線、動画を同時に表示していた。desktopでは一画面に収まる一方、mobileでは最初の候補が長くなり、外部navigationへ進むまでと、ほかの候補を比較するまでのscroll量が大きい。

ADR-0013は動画を見つけやすくするためYouTube iframeの初期表示を採用した。しかし、推薦結果を表示しただけで第三者通信が始まり、動画を見ない利用者にも通信が発生する。Security Baselineの「明示操作後だけ読み込む」という検証項目とも矛盾していた。

## Decision

1. 到着・滑走・終了時刻、推薦理由、料金、受付、情報確認日、access補足、外部navigation、公式情報、公式SNS、訂正報告は推薦cardへ常時表示する。
2. 営業日の注意、一般利用の注意、利用ルール、翻訳fallback、YouTube動画は、施設ごとの`details`要素へまとめる。summaryは日本語・英語で開閉状態を表し、keyboardとscreen readerで操作できる。
3. YouTube iframeは動画を含む施設詳細を利用者が開いた後に初めて生成する。詳細を閉じた場合はplayerを非表示にし、再度開いた場合は同じiframeを再利用する。`autoplay=0`と外部YouTube linkは維持する。
4. 代替候補を一覧表示しただけでは、各候補のYouTube iframeを生成しない。利用者が候補ごとの施設詳細を開いた場合だけ、その候補のiframeを生成する。
5. ADR-0013 Decision 4、初期loadを採用したAlternative、初期表示に伴うNegative consequence、初期iframe生成のVerificationは本ADRで置き換える。ADR-0013のprovider allowlist、URL構築、CSP、外部link、catalog validation、手動選定の判断は維持する。

## Alternatives

### すべての施設情報を常時表示する

追加操作なしで全情報を読めるが、mobileで主要actionと代替候補が遠くなり、比較の負荷が高い。安全性と意思決定に必要な概要を常時表示したうえで補助情報を段階表示する。

### CSSだけでmobile表示を短くする

visualな高さは変えられるが、非表示要素の意味、keyboard操作、YouTube iframeの第三者通信を制御できない。native `details` とJavaScriptのplayer lifecycleを組み合わせる。

### 動画だけを別ページへ分離する

推薦cardは短くなるが、単一pageの比較flowを中断し、routeと状態管理が増える。MVPではcard内の段階表示に留める。

## Consequences

### Positive

- mobileで主要情報とnavigationへ短いscrollで到達できる
- 代替候補を開いたときも、各候補の補助情報を必要な順に確認できる
- 動画を見ない利用者のYouTube第三者通信を避けられる
- Security Baselineと実装・E2Eの通信開始条件が一致する

### Negative

- 注意事項、ルール、動画の確認に1回の追加操作が必要になる
- summary文言と開閉状態を日本語・英語、keyboard、screen readerで検証する必要がある
- 詳細を閉じても生成済みiframe自体はDOMに残るため、通信を取り消すものではない

## Verification

- 推薦直後に主要情報とnavigation linkが表示され、注意事項・ルール・動画が閉じていること
- 推薦直後と代替候補一覧を開いた直後にYouTube iframeが存在しないこと
- 施設詳細を開くと対象施設だけのiframeを1件生成し、閉じると非表示、再表示時は同じiframeを使うこと
- 日本語・英語のsummaryが開閉状態を表し、focus indicatorと44px以上のtap targetを持つこと
- desktopとmobileのE2Eで推薦、外部導線、訂正報告、動画fallbackが継続して動作すること

## Revisit Conditions

- 利用者が注意事項やルールに気付かず、来場判断を誤る事例が確認された
- 施設詳細の展開率が低く、動画導線が目的地決定に寄与しない
- YouTubeのembed、privacy、consent要件が変わる
- 施設cardでは比較できない情報量になり、独立した施設詳細pageが必要になる
