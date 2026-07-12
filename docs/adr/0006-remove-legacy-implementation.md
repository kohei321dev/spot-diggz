# ADR-0006 旧実装を現行ツリーから削除する

- Status: Accepted
- Date: 2026-07-12
- Supersedes: [ADR-0001](0001-repository-strategy.md)の「旧実装を初期段階では削除しない」決定

## Context

旧リポジトリには、当初の地図・スポット投稿サービス向けのiOS、Android、React、Rust API、Firebase、Firestore、Terraform、Cloud Build、運用ドキュメントが混在している。
新しいプロダクトは、初心者・復帰者・来訪者向けのセッション意思決定支援Webサービスとして、要求整理から作り直すことになった。

## Decision

1. 旧iOS、Android、Web、API、インフラ、サンプル、seed／secretスクリプトを現行ツリーから削除する。
2. 旧実装のソースはGitコミット履歴に残し、必要な場合は対象コミットから参照する。
3. 旧実装に依存するCI、Dev Container、Husky、Makefile、旧CLAUDE.md、旧エージェントスキルを削除する。
4. 旧実装専用の設計・運用ドキュメントを削除し、要求、調査、ワークフロー、ADRだけを現行ドキュメントとして残す。
5. 新しいアプリ実装、技術選定、CIは、要求検証後に別のADRとスプリント成果物として追加する。

## Consequences

### Positive

- 現行ツリーが未検証の旧アーキテクチャに引きずられない
- 削除されたコードを誤って拡張するリスクが下がる
- これからの技術選定を要求に合わせて行える
- 旧実装は履歴から復元・比較できる

### Negative

- 旧アプリをすぐに起動することはできない
- 新しい技術構成、CI、開発環境を改めて定義する必要がある

## Verification

- 現行ツリーに旧iOS／Android／Rust／React／GCP実装が存在しない
- `docs/product_baseline.md`、市場調査、ワークフロー、ADRが残っている
- `git log`から削除前のコミットを参照できる
