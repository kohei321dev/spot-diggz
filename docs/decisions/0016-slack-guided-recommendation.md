# ADR-0016 Slack条件入力と推薦応答

- Status: Accepted
- Date: 2026-08-03
- Type: Specification
- Related Issues: Incomplete — repository historyから対応Issueを特定できていない
- Related Pull Requests: [#304](https://github.com/kohei321dev/spot-diggz/pull/304)
- Affected Docs: `requirements.md`, `architecture.md`, `security.md`, `specifications/chat-integrations.md`, `guides/slack-setup.md`
- Supersedes: [ADR-0015](0015-owner-auth-and-chat-entrypoints.md)（Slack入力・response flowのみ）
- Superseded By: None

## Context

固定の代表地点だけで`/spotdiggz`を実行すると、その日の出発地、使える時間、交通手段、目的を反映できず、利用者に最適化された推薦にならない。Slackは最初の応答を短時間で返す必要があり、同じinteractionがretryされる可能性もある。一方、単発の行き先決定に候補保存は不要であり、保存button、参照UI、永続tableを追加すると操作と運用負荷が増える。

## Decision

1. `/spotdiggz`はSlack modalを開き、出発地、利用可能時間、交通手段、レベル、目的、気分を取得する。
2. Slackのraw form body、timestamp、signatureを検証し、設定済みteam/user IDに完全一致するownerだけを許可する。
3. modal submissionへ即時ackした後、地点検索と既存の決定論的推薦をprocess内background taskで行い、最大3件をephemeral messageで返す。
4. 候補には`公式情報`と`ここに行く`だけを表示し、保存操作を設けない。
5. 地点入力、座標、候補facility ID、推薦文、Slack message historyを永続化しない。
6. retry重複処理を防ぐsource event keyと`generating`、`delivered`、`failed`の状態だけをPostgreSQLへ最大1時間保持し、1時間ごとのworkerで期限切れをpurgeする。
7. ProductionでSlackを有効にする場合は`DATABASE_URL`を必須とし、developmentだけmemory storeを許可する。
8. Slack Appは`commands`と`chat:write`だけを要求し、Interactivityを有効にする。DM、mention、Event Subscriptions、Socket Modeは使わない。

## Alternatives

### Slash Commandのtextへ全条件を書く

入力形式の記憶が必要で誤入力しやすく、選択肢の許可値検証も利用者へ分かりにくいため採用しない。

### DMで質問を繰り返す

会話状態、Event Subscriptions、追加scope、message retentionへの依存が増える。MVPの単一owner用途にはmodalで十分なため採用しない。

### 推薦候補をListsへ保存する

単発の行き先決定には不要であり、永続table、保存API、参照UI、削除操作が必要になるため採用しない。

### request状態も保存しない

Vercelの複数instanceや再起動をまたぐSlack retryで処理が重複するため採用しない。個人情報や施設IDを含まない最小状態に限定する。

### 最初からdurable queueを導入する

現時点で必要性を示す障害率・処理量の計測がない。単一deploy単位を維持し、実測で必要になった場合に再検討する。

## Consequences

- 利用者はSlack内で条件を選択し、候補から公式情報または経路へ直接進める。
- 保存button、`GET /api/lists`、`saved_facilities` tableを持たない。
- request tableは重複処理防止用の最小状態だけを最大1時間保持する。
- 正確な地点と候補を永続化しないためprivacyとDB容量のriskを抑えられる。
- processがbackground処理中に停止した場合、地点入力を復元して自動再開できない。

## Verification

- 不正署名、5分超過timestamp、別team/user、oversize bodyを拒否するtest。
- Slash Commandがmodalを開き、modal条件から候補をephemeral送信するtest。
- 候補payloadに保存actionがなく、公式情報と経路actionだけがあるtest。
- source event keyによる重複防止と1時間後のrequest purge test。
- OpenAPI、Manifest、PowerShell構文、secret混入検査。
- Productionではowner/non-owner、modal、候補、公式情報・経路リンクを実資格情報でsmoke testする。
