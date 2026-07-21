# ADR-0004 多言語対応の開始範囲

- Status: Accepted
- Date: 2026-07-12
- Accepted: 2026-07-20
- Related: [Product Baseline](../product_baseline.md)
- Related: [ADR-0002](0002-facility-data-source-and-freshness.md)

## Context

訪日スケーターは、施設の場所だけでなく、利用ルール、ヘルメット、防具、交通、地域のマナーを確認する必要がある。多言語対応は価値になり得るが、全言語を初期対象にすると、source確認、翻訳、UI、test、運用の範囲が過大になる。

施設の公式情報は主に日本語であり、翻訳だけをsource of truthにすると、更新時の差分と誤訳を追跡しにくい。推薦理由は決定論的なcodeを持つため、APIの自由文を再翻訳するより、codeをlocale別の定型文へ対応付ける方が再現可能である。

## Decision

1. MVPの表示言語は日本語と英語に限定する。
2. 施設catalogは日本語のsource-backed情報を正規値とし、英語表示に必要な値を `englishTranslation` へ明示的に保持する。
3. 英訳必須fieldは施設名、住所、営業注意、料金、予約、利用rule、access notesとする。日本語の営業注意・利用ruleと英訳配列の件数を一致させる。
4. 英訳必須fieldが欠けた施設は、起動時catalog検証で公開を拒否する。
5. UI固定文言、入力選択肢、validation、訂正報告はlocale dictionaryで日英を切り替える。
6. 推薦理由は安定した `reason.code` をlocale別定型文へ対応付ける。未知codeを英語で日本語自由文へfallbackして事実を断定しない。
7. 選択したlocaleだけをbrowser local storageへ保存する。正確な現在地、地点検索文字列、推薦条件はlocale設定と一緒に保存しない。
8. AI翻訳はMVP runtimeへ含めない。翻訳案の作成に利用する場合も、公開前にsourceと照合する。
9. 他言語は、日本語・英語の利用者需要と翻訳更新運用が成立した後に追加する。

## Alternatives

### 最初から多言語をすべて実装する

対応可能な利用者は増えるが、翻訳品質、test、support、SEOの負担が大きい。

### 日本語だけで開始する

開発負担は小さいが、訪日スケーターという検証対象を評価できない。

### requestごとに機械翻訳する

catalog作成時の英訳管理を減らせるが、外部依存、cost、latency、表現の揺れ、sourceとの差分を増やす。利用ruleの説明責任を満たしにくいためMVPでは採用しない。

### localeごとに施設record全体を複製する

読み取りは単純になるが、source、検証時刻、座標、features等の言語非依存fieldが重複し、更新漏れが起きやすい。

## Consequences

### Positive

- 日本語・英語の利用者testを同じ推薦ruleで行える
- 施設事実と翻訳の対応をcatalog reviewで確認できる
- locale切替が外部翻訳providerの可用性に依存しない
- translation欠落をrelease前に検出できる

### Negative

- 施設追加と動的情報更新のたびに日英を同時にreviewする必要がある
- `englishTranslation` は日本語fieldと一対一であり、任意言語への一般化をまだ行っていない
- 休場理由等、MVP画面に表示しないfieldの翻訳scopeは別途判断が必要になる

## Verification

- catalogの全施設が完全な `englishTranslation` を持つこと
- 日本語の `scheduleNotes` / `rules` と英語配列の件数が一致すること
- UIのlocale切替で主要入力、推薦結果、注意事項、訂正dialogが日英表示されること
- 英語表示で既知の推薦reason codeが日本語自由文を表示しないこと
- locale以外の位置・検索・推薦条件をbrowser storageへ保存しないこと
- 日英表示で施設source URLと検証時刻が同じ事実を参照すること

## Revisit Conditions

- 第3言語の継続需要が確認された
- 施設数・更新頻度により専用translation workflowが必要になった
- sourceが英語のみの施設を扱う必要が生じた
- runtime翻訳の品質・cost・privacyが静的翻訳より有利になった
