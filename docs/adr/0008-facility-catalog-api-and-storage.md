# ADR-0008 施設カタログの初期APIと保存方式

- Status: Accepted
- Date: 2026-07-16
- Related: [Product Baseline](../product_baseline.md)
- Related: [ADR-0002](0002-facility-data-source-and-freshness.md)
- Related: [ADR-0007](0007-go-modular-monolith-runtime.md)
- Related: [ADR-0009](0009-session-recommendation-ui.md)
- Related: [ADR-0010](0010-google-maps-provider-and-fallback.md)
- Related: [ADR-0011](0011-five-prefecture-mvp-scope.md)

## Context

Sprint 1では、公式情報を確認した施設を登録し、施設詳細を表示できるFoundationが必要である。採択時点では候補23施設の検証済み件数が0件であり、未確認情報をアプリケーションへ混ぜない境界が必要だった。

MVPの施設数は20〜30件、初期トラフィックは小規模である。施設データは高頻度のユーザー書き込みではなく、出典と確認日を伴う運用者の更新が中心である。

## Decision

1. 初期の施設カタログは、Gitでバージョン管理するJSONファイルとして保存する。
2. アプリケーション起動時にJSONを1回読み込み、メモリ上の読み取り専用スナップショットとして提供する。
3. `status=verified`、必須属性、HTTP(S)の`sourceUrl`、`verifiedAt`を起動時に検証する。検証に失敗したカタログではアプリケーションを起動しない。
4. 初期APIは次の読み取り系エンドポイントに限定する。
   - `GET /healthz`: 稼働確認
   - `GET /api/facilities`: 施設一覧。`activity`で競技を絞り込める
   - `GET /api/facilities/{facilityId}`: 施設詳細
5. 施設APIはユーザーの現在地を受け取らず、保存もしない。距離・営業時間による推薦は後続のRecommendation moduleが担当する。
6. APIエラーは安定した`error.code`と利用者向け`error.message`をJSONで返す。

## Implementation Evolution (2026-07-20)

Git管理JSONを起動時に読み取り専用snapshotへloadするDecisionは維持する。その後のMVP実装で次を追加した。

- 公開catalogは5府県31施設（大阪府24施設）となり、`prefecture`、`municipality`、日英必須属性、`dynamicVerifiedAt`、`stableVerifiedAt`、`one_time` / `annual` 休場をschemaと起動時validationへ追加した。`generalUseStatus=schedule_check_required` は施設APIには返すが推薦から除外し、`limited` は安全側の営業時間と注意事項を返す。
- dynamic 30日・stable 180日を超えたrecordは構造errorにせずloadするが、推薦から除外する。fresh recordが1件以上なら `/readyz` は200、emptyまたは全件staleなら503を返す。
- HTTP surfaceはADR-0009/0010とProduct Baselineに従い、readiness、location search、recommendation、correction、product event、metricsまで拡張した。OpenAPIを現在のrequest、response、error codeの正本とする。
- catalog自体は引き続きread-onlyで、訂正reportを自動反映しない。訂正はcatalogとは別storeへ保存し、`DATABASE_URL`設定時はNeon/PostgreSQL、未設定のlocal/CIではJSON Lines file storeを使う。任意contact consent、90日の `deleteAfter`、起動時と1時間ごとのpurgeを適用する。
- image内catalogとcorrection storeを分け、application rollbackでreportを上書きしない。Vercel Productionのcorrection storeはNeonを使う。

本ADRのContextと「空のcatalog」という記述は採択時点の履歴であり、現在件数とrelease条件はProduct Baselineを正とする。

## Alternatives

### PostgreSQLを最初から採用する

将来の運用者更新、履歴、複数出典、検索には適する。一方、検証済みデータが0件で更新UIも未定義の現段階では、DB・migration・バックアップ・接続障害という運用対象を先に増やす。必要性が確認できた時点で移行する。

### 未確認の候補をAPIで返す

画面開発を早められるが、営業時間や利用可否を誤って断定する。Product BaselineとADR-0002の出典・鮮度方針に反するため採用しない。

### 外部施設APIをランタイム参照する

初期データ作成を減らせる可能性はあるが、スケートボード利用可否、初心者適性、ルール、出典の責任を満たせない。外部APIは必要な属性と利用規約を確認した後に個別評価する。

## Consequences

### Positive

- 同じGit commitから同じカタログとGo binaryを再現できる
- 未確認データの公開を起動時に拒否できる
- DBや外部APIなしでローカル・CI・コンテナの動作確認ができる
- 読み取り専用APIの責務とRecommendation moduleの責務を分けられる

### Negative

- 施設更新にはJSON変更とデプロイが必要になる
- 運用者向け編集画面、変更履歴、複数人の同時更新は提供しない
- 施設数、更新頻度、検索要件が増えた場合はDB移行が必要になる
- 施設情報は公式ページの更新、臨時休場、貸切、天候に影響されるため、公開後も再確認運用が必要になる

## Verification

- JSONの必須属性、重複ID、座標範囲、出典URL、確認日、`status`を単体テストで検証する
- 一覧、競技絞り込み、詳細、404、healthz、安定エラーコードをHTTPテストで検証する
- CIでformat、vet、test、build、Go脆弱性検査、コンテナ検査を実行する
- 20〜30件の検証済みデータを登録後、施設確認・更新時間が週2時間以内かSprint Reviewで確認する

## Revisit Conditions

- 運用者更新の頻度や履歴管理がJSON運用の負荷を超える
- 複数プロセス・複数リージョンでの更新整合性が必要になる
- 施設数または検索条件が増え、メモリスナップショットの読み込み・検索が問題になる
