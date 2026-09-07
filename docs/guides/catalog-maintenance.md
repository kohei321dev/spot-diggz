# カタログの確認・更新手順

- Status: Current procedure; production data re-verification incomplete
- Audience: owner / カタログ保守担当
- Specification: [施設データ](../specifications/facility-data.md)
- Decision: [DR-0022](../decisions/0022-domestic-catalog-quality.md)
- Tracking: [#314](https://github.com/kohei321dev/spot-diggz/issues/314)、公開gateは#318

## 責任と対象

ownerが確認担当とreview担当を決め、Git管理の`data/facilities.json`をPRで更新します。現在はownerが保守責任を持ちます。APIは起動時snapshotを読むだけで、外部検索・利用者入力・AI結果を自動登録しません。検索本文・現在地・個人情報・非公開URLを確認記録へ含めません。

国内47都道府県の正式名称を受理しますが、収録済みは2026-07-19調査基準の5府県31施設です。地域名の許可、実際の収録、最新の検索掲載可否は別です。#314では実recordのgenre・確認日・件数を変更していません。

## 更新フロー

1. remote mainと関連Issue/PRを確認し、施設ID・出典・変更目的を特定します。既存IDを振り直さず、同一施設の重複登録を避けます。
2. 自治体・管理者・施設の公開公式情報で所在地、競技、滑走許可、利用ルールを確認します。動画に滑走者がいることやパーク内のstreet sectionを`genre=street`の根拠にしません。
3. 都道府県・市区町村・住所・公開座標の一致を確認します。validatorは正式名称と座標の有限性・範囲を検査しますが、実際の行政区域との一致までは証明しません。海外・時差対応は別の契約変更です。
4. 次表でhoursStatusを選びます。非該当の根拠は公開HTTPSの`hoursSourceUrl`、滑走許可の根拠は`skatingPermissionSourceUrl`に分けて記録します。同じ公式ページが両方を裏付ける場合は同じURLで構いません。URLには認証情報や非公開ページを使いません。
5. 料金・予約・登録・安全ルール等も確認し、日英の意味を一致させます。必須の事実が不足する候補は公開catalogへ昇格させません。
6. 再確認した属性群に限り確認時刻をタイムゾーン付きRFC 3339で記録します。`dynamicVerifiedAt`は営業時間状態・許可・料金・予約・ルール等、`stableVerifiedAt`は名称・所在地・設備・access等です。`verifiedAt`も実際の確認を表し、日時だけの一括更新で鮮度を偽装しません。
7. PRへ確認対象、公開出典、変更した事実、担当、確認日、未確定事項、検証結果を簡潔に記録します。raw provider responseやページ全文を転載しません。構造検証・内容review後も公開反映は別途承認します。

## 営業時間を決める

| 確認した事実 | 設定 | 掲載判断 |
| --- | --- | --- |
| 曜日別の利用時間が分かる | `known`と有効な`hours`。旧recordのstatus省略もこの扱い | 他の品質条件を満たせば検索可。旧旅程計算では営業時間・休場等も評価 |
| 営業時間制度が適用されないことを公式情報で確認したstreet | `not_applicable`、hoursSourceUrl、明示的regular/limited、日英availabilityNote。hours/hoursBasisなし | 他の品質条件を満たせば周辺検索可。今滑れる/24時間利用可とは言わない。旧旅程計算から除外 |
| 時間の情報が見つからない・確認できない | `unknown`、schedule_check_required、日英availabilityNote。hours/hoursBasisなし | その他必須事実を確認済みなら詳細参照のみ。検索・旧旅程計算から除外 |

非knownのcatalog入力では`hours: []`と`hoursBasis: ""`も受理しますが、canonical recordでは省略を使います。API responseも省略するため、Bot/clientは欠落を無制限と表示せず、状態と説明を表示します。

営業時間制度が非該当である確認記録は次の形式で理由と再確認条件も残します（実施設の根拠を入力し、例をそのまま事実として使いません）。

```markdown
- Status: Not applicable
- Reason: <管理者のどの公開根拠から営業時間制度の対象外と確認したか>
- Revisit when: <管理規則・入場方式・時間制限の変更時>
```

単に不明な候補は以下を使います。

```markdown
- Status: Incomplete
- Missing evidence: <未確認の許可・料金・時間等>
- Required decision: <担当者が確認する出典や管理者への確認事項>
```

候補の保存先は[調査資料](../research/README.md)です。新しいDB・状態store・自動収集を追加する手順ではありません。

## 検証と公開前gate

```text
go test ./internal/facility ./internal/nearby ./internal/httpapi ./internal/recommendation ./cmd/catalogcheck
npm run test:contracts
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-docs.ps1
git diff --check
go run ./cmd/catalogcheck -require-searchable
```

最後のコマンドだけは実行時刻の本番catalogを検査します。既存の全recordのdynamic 30日・stable 180日を168時間先まで検査したうえで、その終端にも検索掲載可能なrecordが1件以上必要です。期限直前なら失敗し、分類・hours状態・一般利用条件が不足して全件対象外でも失敗します。検査はread-onlyで、日付やgenreを変更しません。検証期間を短縮して通常の公開gateを回避しません。

通常CIは固定clockの合成fixtureによる回帰検証です。成功しても実catalogの確認・実Google疎通・owner/non-owner E2E・Cloud Run公開の成功を意味しません。`data_unavailable`やreadiness 503を実在スポット0件と解釈せず、出典の再確認と後続データ整備へ戻ります。

## 定期保守とrollback

ownerは週次および公開前にgateを実行し、期限が迫る属性群を再確認します。変更された許可・営業時間・ルールは日英説明と一緒に更新します。根拠不足や休止の疑いがあるrecordを日時だけ新しくして検索へ戻しません。必要な公開除外は別PRで根拠と移行先を残します。

code・OpenAPI・catalogの検証済み組を保持します。新fieldや旧5府県外のrecordを含むcatalogは旧binaryが拒否する場合があるため、rollbackは互換な組へ戻します。新recordを物理削除して旧validatorに合わせたり、DB・利用者データを巻き戻したりしません。再起動後に認証付き検索・詳細とreadinessを確認します。

- Status: Incomplete
- Missing evidence: 実catalogのgenre根拠・現在の鮮度再確認、公開先での検索結果と保守工数。
- Required decision: ownerが後続の実データ確認範囲・担当・公開時期を決め、#318の公開gateと合わせて確認する。
