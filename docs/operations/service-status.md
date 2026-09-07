# 公開先と旧資料の整理状況

- Status: Incomplete
- Last checked: 2026-09-06 (JST)
- Baseline: remote main `c30e73df1875ae0048b3949c6ab2da527393130b`
- Missing evidence: Cloud Runの実project・公開URL・API認証/Gateway・Bot連携・予算メール設定と稼働確認。
- Required decision: Cloud Run・独自Web UI不要・API/Bot提供方針はDR-0019で採用済み。#318で実際の環境・費用・認証・通知設定を確認し、外部変更とdeployの明示承認後にsmoke結果を記録する。

## 新構成の設定状況

[Cloud Run運用計画](cloud-run-plan.md)に月額目安USD 3・Google Cloud予算メール通知・超過時停止なしを記録しました。通知設定、Gateway、Cloud Runへの移行・公開はいずれも未実施です。以下のHTTP確認は過去の調査記録であり、今回再測定したものではありません。

## 確認した事実

- 従来案内していたhost `spotdiggz.vercel.app`の`/`と`/healthz`へ認証情報なしのGETを行い、両方でHTTP 404を確認しました。本文、Cookie、秘密情報は保存していません。
- HTTP 404だけでは、Vercel Project削除、停止、移転、設定変更のどれかは断定できません。
- リポジトリにはGo製Web/API、owner認証、Slack/Discord、Vercel設定が残っています。ソースの存在は本番稼働の証拠ではありません。
- 過去のProduction確認記録は確認当時の履歴であり、現在アクセスできることを意味しません。

## 設定手順の実行前条件

各guideの`https://<deployment-host>`は実行用URLではなく、確認済みHTTPS originへの置換箇所です。公開先が未確認の間はOAuth callback、Slack/Discord endpoint、secret登録、deployを実行しません。

既存の`slack-manifest.json`、`scripts/check-integration-cli-prerequisites.ps1`、`scripts/configure-slack-vercel-env.ps1`には従来hostの固定値があります。`scripts/configure-github-vercel-env.ps1`のBaseURL既定値も従来hostです。これらは新しい公開先の確認後に整合させる必要があり、現在有効な設定としてコピー・実行しません。この文書整理ではruntime設定や外部環境を変更していません。

## 削除・訂正の根拠

| 対象 | 整理内容 | 根拠・引継ぎ先 |
| --- | --- | --- |
| `docs/research/ui/facility-card-video-first-sample.html` | 削除 | iframe初期読込・独立詳細へのリンクが現行と異なる。正本は[Web UI仕様](../specifications/web-ui.md)とADR-0014 |
| `docs/research/ui/facility-detail-video-first-sample.html` | 削除 | 独立詳細・一覧への導線を持つ旧案。現在の詳細は推薦card内で開く |
| `docs/research/ui/facility-list-sample.html` | 削除 | 現在提供しない一覧案。V2 Issue #294〜#300は現行計画の対象外としてClose済み |
| `docs/research/ui/README.md` | 削除 | 上記sampleへの入口。参照先はWeb UI仕様へ統一 |
| README・guideの旧公開URL | 現役のアクセスリンクを削除し、公開先未確認と明示 | 本書のHTTP確認。設定例は確認済みoriginへの置換が必要 |
| 利用guideの一覧表示・Google未設定断定 | 訂正 | `internal/webui/handler.go`、`internal/geocoding`、`internal/travel`の観測した実装 |
| 観測性文書のmedia未実装・file storeのみの説明 | 訂正 | `internal/observability/registry.go`、`internal/httpapi/server_test.go`、`internal/correction/postgres_store.go` |

## 保持する履歴と復元

Accepted/Superseded Decision Recordの本文、施設情報の出典となる調査snapshot、過去の初期計画、旧Issue参照用のcompatibility stubは現行UIの案内ではないため保持します。

削除したsampleは基準commitのGit履歴から参照・復元できます。画面の根拠には使用しません。今回の整理commitをrevertすれば文書を復元できますが、旧URLや非提供UIを現役として案内しないよう再確認してください。
