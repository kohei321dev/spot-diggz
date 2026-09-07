# DR-0020: メンションと場所名による周辺スポット検索をMVPにする

- Status: Accepted
- Date: 2026-09-07
- Type: Specification
- Related Issues: [#312](https://github.com/kohei321dev/spot-diggz/issues/312)、#313〜#316、#318
- Related Pull Requests: [#320](https://github.com/kohei321dev/spot-diggz/pull/320)
- Affected Docs: `product.md`, `requirements.md`, `specifications/nearby-search.md`, `specifications/api-mvp.md`, `specifications/chat-integrations.md`, `specifications/facility-data.md`, `architecture.md`, `security.md`, `guides/how-to-use.md`, `research/api-client-contract-plan.md`
- Supersedes: ADR-0009の6条件必須・即時滑走推薦/最大3件を新MVPの基本とする範囲、ADR-0016のslash/modal入口・mention非対応と固定候補数、DR-0019の入力/検索の範囲のみ。認証・保存・品質・ホスト・予算方針は維持。
- Superseded By: None

## Context

ownerは場所・エリア・目的地から探す用途を求め、入力/出力と利用者ユースケースを実装前に整理するよう依頼した。「探す・確認する」を基本にし、移動時間を含む判断を追加条件の機能へ分ける方向に合意した。その後、MVPをメンションと場所名だけで周辺スポットを返す形に絞り、genre・limit・sortの任意指定を承認し、検索範囲も指定できる方向を提示した。

基準remote mainは`b22fd4035c3aaf039955cfa8fdd226b5e0328e44`（PR #319のマージ）。本記録は今回の合意を文書化し、技術詳細やruntime実装済みの証拠とはしない。

## Decision

1. MVPの利用者入力は`@SpotDiggz [場所名]`。任意に`genre`、`limit`、`sort`を付けられる。場所は検索基準であり、現在地/出発地を必須にしない。
2. `genre=skatepark`と`genre=street`を区別し、省略時は両方。streetはパーク外の街中にある滑走可能なスポットであり、パーク内のstreet sectionとは区別する。
3. `limit`で最大候補数を指定し、`sort=distance`で基準地点からの直線距離が近い順に返す。MVPのsortはdistanceだけとし、省略可能にする。
4. 検索範囲も任意指定可能にする。`radius=10km`は構文案で、数値既定値・上下限等は別途確定する。limitと範囲を混同せず、条件を自動緩和しない。
5. 場所の複数候補は利用者へ確認し、検索中→結果/エラーの流れを保つ。未登録を不存在と断定せず、未検証の事実を生成しない。
6. streetは滑走可能な根拠と利用ルールが確認されたものだけを扱う。実スポット追加は後続とし、今回本番catalogを書き換えない。
7. 目的別mode、自然文の意図理解、料金/おすすめsort、移動・到着・滑走時間の算出を今回の周辺検索MVPへ追加しない。詳細操作とresponseの厳密な契約は提案として扱う。

採用範囲と未確定値の正本は[周辺検索仕様](../specifications/nearby-search.md)。limit省略時5件・1〜10件、radius構文、返却field案を数値/schemaまで承認済みとして実装しない。

## Rationale

場所名だけで最初の候補を得られ、必要な人だけ条件を指定できる。近い順は根拠を説明しやすく、移動計画に必要な追加情報を基本検索へ強制しない。APIとBotの責務を分け、入口ごとに施設判断を複製しない。

## Alternatives Considered

- 6条件modalを必須にする: 周辺を知りたい利用者に不要な入力を求めるため、新MVPの基本入口としては採用しない。
- 複数modeを最初から実装する: 各用途の応答/検証範囲が増えるため後続拡張とする。
- 未確認streetを候補に加える: 滑走可能性を根拠なく保証するため採用しない。
- 検索半径を固定する: 場所に応じて範囲を指定したい今回の方針を満たさない。

## Consequences

旧6条件・固定3件・slash/modal非mention契約を新入力の正本から外し、旧仕様は移行前の参照として保持する。#312の構文/応答・安全境界、#313の検索API、#314のgenre/品質、#315/#316のメンション入口、#318のBot配置/費用検証へ反映する。PR #319のマージだけでこれらの詳細が完成したとは扱わない。

## Security and Privacy

owner限定、検索位置・本文・候補・tokenのapplication非保存、履歴非依存を維持する。メンション入力はチャンネルへの結果公開や無条件なtransport変更の承認ではない。platform別の受信・真正性・owner限定返信・停止/配送不能を検証し、旧interaction契約を検証なしで流用しない。

## Migration and Rollback

今回はdocsとIssueの整合だけ。既存ADRの本文・ID・日付を保持し、部分置換metadataを追加する。runtime、API schema、catalog、外部設定は変えない。文書変更はcommit単位でrevertし、Issue本文は編集履歴と更新前控えから復旧する。採用方針そのものの変更には新しいDRを作る。

## Validation

文書構造・内部link・DR metadata/ID、既存JSON/OpenAPI検証、git diff --checkを実行する。周辺検索の後続受入testは仕様の[検証項目](../specifications/nearby-search.md#後続実装の検証項目)で追跡し、文書検証を実機能の成功としない。

## Revisit Conditions

数値/構文・response・genre移行・検索の品質条件、platform別メンション/返信・認証・非同期方式を確定するとき。mode、経路計算、保存または公開範囲を拡張するとき。

## References

- [MVP API契約](../specifications/api-mvp.md)
- [DR-0019](0019-api-client-boundary.md)
- [ADR-0016](0016-slack-guided-recommendation.md)
