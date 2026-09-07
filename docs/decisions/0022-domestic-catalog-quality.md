# DR-0022: 国内の地域表現と営業時間の確認状態を分離する

- Status: Accepted
- Date: 2026-09-07
- Type: Specification
- Related Issues: [#314](https://github.com/kohei321dev/spot-diggz/issues/314)、#312、#313、#318
- Related Pull Requests: [#322](https://github.com/kohei321dev/spot-diggz/pull/322)
- Affected Docs: `specifications/facility-data.md`, `specifications/read-api.md`, `specifications/facility-catalog.openapi.yaml`, `requirements.md`, `product.md`, `security.md`, `guides/catalog-maintenance.md`, `operations/mvp-runbook.md`
- Supersedes: ADR-0011の5府県validator制約、DR-0021後続事項の地域表現・streetの営業時間未定義表現のみ。既存record本文・ID・収録履歴は保持。
- Superseded By: None

## Context

PR #321はremote main `40b2157422446c909210c7ebe42117917fead930`へマージされた。ownerから新計画の継続実装を依頼され、#314を次の検証可能な単位とする。対象利用者は地域非限定だが、既存validatorは5府県のみで、営業時間が存在しないstreetを追加するには架空の時刻を埋める必要があった。

以下は#314の範囲内でagentが具体化した実装判断であり、全国の実施設確認、海外対応、外部公開までownerが承認したと解釈しない。事実の裏付け・非保存・認証・費用方針は維持する。

## Decision

1. `prefecture`は日本の47都道府県の正式名称を完全一致で受理する。略称・任意文字列・海外州名は受理しない。`municipality`と公開施設住所は必須を維持し、行政区域との実一致はcuratorが出典で確認する。国内Geocoding/JSTという既存契約に合わせた実装範囲であり、ターゲットを日本に限定するProduct定義ではない。
2. 加算的な`hoursStatus`を`known` / `not_applicable` / `unknown`で表す。旧recordの省略はknown相当で、従来どおり有効なhoursを必須にする。省略から営業時間非該当を推測しない。
3. `not_applicable`はstreetだけに認める。営業時間制がないことを確認したHTTPSの`hoursSourceUrl`、日英`availabilityNote`、明示的regular/limitedの一般利用状態、滑走許可の根拠・ルールを必須にする。hoursとhoursBasisは空/省略とし、24時間営業の架空値に置き換えない。
4. `unknown`は分類済みpark/streetの参照情報として保持できるが、hoursは空/省略、一般利用状態はschedule_check_required、日英の不足説明を必須にする。未確認施設のstatusをverifiedへ昇格する意味ではなく、存在・出典等は従来の検証条件を満たす必要がある。新検索と旧即時滑走推薦から除外する。
5. 新周辺検索は「知られている場所を比較する」ものであり、根拠のあるnot_applicableは検索対象にできるが「今滑れる」と保証しない。旧到着/滑走時間計算はknown以外をprovider呼出し前に除外する。検索・readiness・運用検査は同じ掲載条件を参照する。
6. 料金・予約・ルール・日英の案内は必須を維持する。不明を無料/不要に変換しない。営業時間以外の重要情報が不足して判断できないrecordは公開catalogへ昇格させず、候補資料でIncompleteとして保持する。
7. 本番catalogのgenre・確認日は変更せず、出典の再調査/実スポット追加は後続とする。未分類recordは引き続き新検索から除外。更新手順には担当、出典、確認時刻、日英、鮮度、退役/rollbackを含める。
8. `catalogcheck -require-searchable`を公開前の追加gateにする。全recordが指定期間freshである既存検査に加え、同期間の終端でも掲載可能なスケートボードrecordが1件以上あることを確認する。通常CIは固定fixtureで検証し、本番の確認日だけ更新して成功させない。

## Alternatives

- 5府県のまま: 地域非限定の方針と#314の整合が進まず不採用。
- 世界共通の任意住所/国/タイムゾーンへ同時拡張: 既存国内Geocoding/JSTを超え、根拠と仕様が不足するため後続。
- hours空をすべて24時間利用可とする: 不明と対象外を混同するため不採用。
- streetすべてを営業時間非該当とする: 時間制限がある場所も推測で無制限にするため不採用。
- unknownを自動で候補へ含める: 訪問可否の根拠不足を解消しないため不採用。
- 既存recordを一括で分類/日時更新する: 事実確認を伴わないため不採用。

## Consequences

旧JSONはそのまま読み込めるが、新fieldや旧5府県外recordを取り込んだ成果物は旧binaryが拒否する可能性がある。schema・code・catalogを同じversionで配備し、rollbackは互換なsnapshotの組で行う。旧validatorへ合わせるためにrecordを物理削除する運用はしない。

新状態の情報が届くAPI clientは、hours欠落を無制限/無料と表示せず、hoursStatusとavailabilityNoteを表示する。現在の本番catalogは変えないため、このPRだけで有効な候補が増えるとはしない。

## Verification

- 47名称、旧5府県外、略称/空白/無効地域、有限座標、既存JSONの互換性。
- known/非該当/不明とhours/根拠/genre/一般利用/日英noteの整合、不正URL・不足ruleの拒否。
- fresh/stale/未来日時/未検証status、unknown・日付確認必須・未分類の検索除外、not_applicableの周辺検索と旧推薦非混入。
- `catalogcheck`の旧動作、新gateの0件/未分類/未知時間/非該当/期限境界、HTTP responseとreadiness。
- Go format/vet/race/test/build、OpenAPI/文書検証、既定CI、diff/secret確認。

## Revisit Conditions

海外対応、住所/timezone構造の変更、営業時間以外の不明属性を公開する需要、正規の許可/時間情報を取得できないstreet、実データ追加、更新工数の実測時。

## References

- [e-Stat 市区町村を探す（都道府県一覧）](https://www.e-stat.go.jp/municipalities/cities/areacode)
- [DR-0021](0021-read-api-contract.md)
- [#314](https://github.com/kohei321dev/spot-diggz/issues/314)
