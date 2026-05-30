---
name: verify
description: CI相当のチェック（fmt/clippy/test/lint/type-check/build）をローカルで全て実行
allowed-tools: Bash(cargo fmt:*), Bash(cargo clippy:*), Bash(cargo test:*), Bash(npm run lint:*), Bash(npm run type-check:*), Bash(npm test:*), Bash(npm run build:*), Bash(terraform fmt:*), Bash(terraform validate:*)
---

CI相当のチェックをローカルで全て実行し、結果を報告してください。

実行順序:

1. Rust API チェック
   ```
   cd web/api
   cargo fmt -- --check
   cargo clippy -- -D warnings
   cargo test --verbose
   ```
2. React UI チェック
   ```
   cd web/ui
   npm run lint
   npm run type-check
   npm test -- --coverage --watch=false
   npm run build
   ```
3. Terraform チェック（web/resources/ に変更がある場合のみ）
   ```
   cd web/resources
   terraform fmt -check -recursive
   ```

結果サマリ:

- 全PASS → 「CI Ready」と表示
- 失敗あり → 失敗箇所と修正案を提示
