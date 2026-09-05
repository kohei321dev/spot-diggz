# 公開先と旧資料の整理状況

- Status: Incomplete
- Last checked: 2026-09-06 (JST)
- Baseline: remote main `c30e73df1875ae0048b3949c6ab2da527393130b`
- Missing evidence: 現在提供中の公開URL、Web/APIの提供継続・廃止・移転の判断、外部連携の稼働状態。
- Required decision: ownerが現在の公開先と提供範囲を確認し、再提供する場合は認証・連携設定とsmoke結果を記録する。

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
