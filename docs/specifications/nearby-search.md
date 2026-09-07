# 場所名からの周辺スポット検索

- Status: API subset implemented in source; Bot contract/connection incomplete
- Last reviewed: 2026-09-07
- Decision: [DR-0020](../decisions/0020-mention-nearby-search.md)、[DR-0021](../decisions/0021-read-api-contract.md)

検索APIの認証・JSON・数値・応答・品質条件は[読み取りAPI](read-api.md)が正本です。以下のメンション例はBot未実装のため実行できません。APIのソース実装と本番公開を区別します。
- Tracking: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、#313〜#316、#318

## 利用者とMVPの目的

利用者はスケボーをしたい人。private MVPでは許可済みownerがSlack/Discordで場所名を指定し、その周辺の登録済み・検証済みスポットを比較する。直接のAPI利用者はBot/appであり、APIは構造化データ、Botは入力解釈とchat表示を担当する。独自Web UI、保存Listsは設けない。

「探す・確認する」を基本とし、移動時間を含む「行けるか判断する」は追加条件を伴う後続拡張に分ける。今回のMVPは周辺検索に絞り、詳細取得の新しい操作モードや複雑なキーワード・目的別検索まで同時実装しない。既存詳細APIの廃止を意味しない。

## 採用した入力

```text
@SpotDiggz [場所名] [genre=種類] [limit=件数] [sort=並び順]
```

場所名だけで使い、必要な利用者だけ任意オプションを付ける。以下は将来の利用例であり、現在実行できるcommandではない。

```text
@SpotDiggz 渋谷駅
@SpotDiggz 渋谷駅 genre=street limit=3
@SpotDiggz 大阪駅 genre=skatepark sort=distance
```

| 入力 | 採用した意味 | 詳細の状態 |
| --- | --- | --- |
| 場所名 | 周辺検索の基準地点。出発地や現在地とは別の概念 | 必須。地名・駅名・施設名の曖昧性を解消し、複数候補の先頭を自動採用しない |
| `genre` | `skatepark`はスケートパーク、`street`はパーク外の街中にある滑走可能なスポット | 任意。省略時は両方。パーク内のstreet sectionや既存`purpose=street`とは別の分類 |
| `limit` | 最大候補数。実際の候補が少なければ、その件数だけ返す | 任意。API初期値5件、許可1〜10件（DR-0021）。Botの文法は後続 |
| `sort` | `distance`で基準地点からの直線距離が近い順 | 任意。MVPでは`distance`のみで、省略時も同じ。料金順・おすすめ順は後続 |
| 検索範囲 | 利用者が任意指定できる | APIは`radiusKm`（km、小数可）、既定10、0.1〜50。Botの`radius=10km`構文は後続 |

検索範囲を含む構文案:

```text
@SpotDiggz 渋谷駅 genre=street limit=3 sort=distance radius=10km
```

APIの既定半径はDR-0021で10kmとした。上記はBot構文案であり実行可能なcommandではない。`limit`は件数、検索範囲は地理条件として分離し、件数を増やすために検索範囲やgenreを自動緩和しない。

出発地、現在地取得、気分、経験レベル、移動手段、滑走時間は基本検索の必須入力にしない。旧6条件へ架空の既定値を補って周辺検索の代用にしない。到着時刻・移動時間・滑走時間を直線距離だけから生成しない。

未知のoption/enum、不正数値、範囲外の値を黙って無視せず、利用できる形式と次の操作を案内する。重複option、空白を含む場所名、引用符、option順序、場所名欠落の厳密な文法とerror codeは#312で確定する。

## 検索と返信

1. 明示的メンションを入口とし、platformで真正性・許可済みownerを検証する。
2. 検索中を表示する。場所を特定できない場合は再入力、複数候補がある場合は利用者の選択へつなげる。確認の具体UIは未確定。
3. Botから認証付きSpotDiggz APIへ場所とoptionを構造化して渡す。`POST /api/facilities/search`がJSON入力と内部場所解決を担当する。Bearer認証と応答は[読み取りAPI](read-api.md)を参照。
4. 地理範囲・genre・品質条件で候補を絞り、直線距離順で最大`limit`件を返す。APIでは丸め前の直線距離kmで境界を含め、同距離はfacilityId昇順。
5. 正常0件、収録データ不足、根拠不足、処理障害を区別する。未登録を「その地域にスポットが存在しない」と表現せず、推測で補完しない。

APIは`matches[].facility`と`distanceKm`、適用条件、注意文、構造化statusを返す。詳細は既存Facility schemaを利用し、Botの表示要約とは分離する。曖昧地点はラベルだけ返し、queryを具体化して再入力する。詳細選択buttonや新しい`detail`モードは今回の実装に含めない。

## スポット品質

`street`は滑走可能な根拠と利用ルールを確認できた場所だけを対象にする。「滑っている人がいる」「障害物がある」だけで掲載せず、立入・滑走の可否が不明な場所は推薦しない。genreの追加は無審査投稿、scraping、実スポット一括追加を認めるものではない。

既存の出典・確認時刻・鮮度・利用可否の品質原則を維持する。ただし、現在の営業時間/到着時刻フィルターを日付未指定の周辺検索へそのまま適用しない。DR-0021ではgenre/滑走根拠/鮮度不足と日付別の一般利用確認が必要なrecordを除外する。休場・時間外は現在時刻でfilterせず、施設情報と注意文で訪問前確認を求める。「今滑れる」は保証しない。streetに営業時間等が存在しない場合のschema表現と既存recordのgenre移行は未確定であり、情報欠落を無条件利用可にしない。

## 安全境界と実装開始前の確認

- owner限定、本文・検索位置・候補・返信tokenのapplication非保存、履歴API非依存を維持する。入力messageがchatサービス側に存在することと、SpotDiggzが履歴を保存しないことを区別する。
- メンションは入力方法の合意であり、Slack/Discordの受信transport、scope/intent、接続維持、再送対策、本人だけへの返信方式の選定・動作確認はまだ行っていない。旧slash command用の署名・ACK・ephemeral方式をそのまま利用できると断定しない。
- 通常messageの履歴巡回を追加せず、Bot自身の返信で再起動するループ、偽メンション、非owner要求を拒否する設計を#315/#316へ引き継ぐ。チャンネル全体への結果公開を今回承認したとは扱わない。
- R-008の適用はDR-0021で新APIに限定して明確化した。検索中心座標・元queryを返さず、曖昧地点の確認ラベルだけ認証済みclientへ返す。施設の公開座標は別情報。旧地点APIの退役とBotの表示範囲は後続とし、位置の保存・公開を追加しない。

- Status: Incomplete
- Missing evidence: Botの構文詳細・platform別メンション受信/owner限定返信・非同期処理と配送失敗対策、streetの営業時間未定義表現、既存genre移行、旧Web退役、公開環境。API側の数値・応答・認証はDR-0021で確定。
- Required decision: agentが残るBot/保存/運用契約を#312/#314〜#316/#318で具体化する。採用済み入力と読み取りAPI契約は再承認を要求しない。

## 後続実装の検証項目

- 場所名だけ、各option単独/組合せ、genre両方/各種類、任意範囲と件数を独立して検証する。
- 不正値・未対応sort・重複option・曖昧地点・場所未特定・0件・未収録/情報不足・provider障害を区別する。
- 指定範囲外を含めず、limit超過・自動範囲拡張をせず、固定fixtureから距離順で再現可能な結果を返す。
- streetの利用可否/根拠不足、期限超過、parkのstreet sectionとの混同を検証する。実施設追加の代わりに架空fixtureを本番へ投入しない。
- owner/非owner、真正性、重複event、Bot返信ループ、配送不能、本文/位置/token非保存を検証する。
- 検索APIはruntime・OpenAPI・testを整備した。実catalog、manifest、実資格情報、deployは変更していない。
