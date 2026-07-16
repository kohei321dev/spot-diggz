# ADR-0005 mainブランチへの移行

- Status: Accepted
- Date: 2026-07-12
- Supersedes: `product/session-planner`を作業ブランチとする運用

## Context

プロダクト要求と開発ワークフローの整理が完了し、`product/session-planner`で作成した内容を今後の標準状態として扱う段階になった。
既存実装はGitのコミット履歴から参照できるため、今回の方針整理だけを理由に永続的な退避ブランチを維持する必要はない。

## Decision

1. `product/session-planner`の変更をコミットしたうえで、正式な統合ブランチを`main`に移行する。
2. 既存の`master`のコミット履歴は、`main`の祖先履歴として保持する。
3. 今回の移行作業で作成した一時的な`archive/pre-session-planner-20260712`は、移行完了後に削除する。
4. 今後の機能開発は`main`から短命なfeatureブランチを作成して行う。
5. GitHub上のデフォルトブランチ変更やリモートブランチ削除は、ローカル移行とは分けて明示的に実施する。

## Consequences

### Positive

- 現行プロダクトの基準ブランチが`main`に統一される
- 旧実装をGitのコミット履歴から参照できる
- 不要な退避ブランチの維持コストを減らせる

### Negative

- GitHubのデフォルトブランチ設定やCIの実行先は、必要に応じて別途確認する必要がある
- 旧実装を頻繁に比較する場合は、コミットIDやタグを記録する必要がある

## Verification

- `main`が現行プロダクト方針のコミットを指していること
- `main`の履歴から、方針変更前のコミットを参照できること
- `archive/pre-session-planner-20260712`が不要になったこと
