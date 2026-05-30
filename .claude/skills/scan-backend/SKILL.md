---
name: scan-backend
description: バックエンドレイヤーのセキュリティスキャン（Rust依存関係, Docker base image, OpenSSL）
allowed-tools: Bash(cargo audit:*), Bash(cargo outdated:*), Bash(gh issue:*), Bash(gh project:*)
---

バックエンドレイヤーのセキュリティ・バージョンスキャンを実行してください。

現在のDockerfile設定:

- ビルドステージ: `rust:1.90-bullseye`
- ランタイムステージ: `debian:bullseye-slim`
- ランタイム依存: `ca-certificates`, `libssl1.1`

手順:

## 1. Rust依存関係の監査

```
cd web/api
cargo audit
cargo outdated
```

1. `cargo audit` の結果からCVE一覧を抽出し、severity（critical/high/moderate/low）で分類
2. `cargo outdated` の結果からパッチ・マイナー・メジャー更新可能なクレートを分類

## 2. 認証・セキュリティ関連クレートの重点確認

以下のクレートについて、`cargo audit` の結果に加えてWebSearchで個別に最新のCVE情報を確認:

| クレート               | 用途                                 | 重要度       |
| ---------------------- | ------------------------------------ | ------------ |
| `jsonwebtoken`         | JWT検証（Firebase IDトークン）       | 認証コア     |
| `reqwest`              | HTTP クライアント（外部API呼び出し） | ネットワーク |
| `base64`               | Base64エンコード/デコード            | 暗号処理補助 |
| `serde` / `serde_json` | シリアライゼーション                 | データ処理   |
| `uuid`                 | UUID生成                             | ID生成       |

1. WebSearch で「{クレート名} rust CVE vulnerability 2025 2026」を検索（認証関連を優先）
2. 脆弱性が見つかった場合、CVSS スコアと影響範囲を記録

## 3. Docker base image 脆弱性確認

1. WebSearch で「rust 1.90 docker image vulnerability CVE 2025 2026」を検索
2. WebSearch で「debian bullseye-slim vulnerability CVE 2025 2026」を検索
3. WebSearch で「debian bullseye EOL end of life date」を検索し、EOL状況を確認
4. Debian Bullseye（11）のサポート期限と現在の状態を報告

## 4. libssl1.1 脆弱性確認

1. WebSearch で「libssl1.1 openssl vulnerability CVE 2025 2026」を検索
2. WebSearch で「openssl 1.1.1 EOL end of life」を検索
3. OpenSSL 1.1.1のサポート状況と後継（OpenSSL 3.x）への移行推奨を確認

## 5. Rust最新stable確認

1. WebSearch で「Rust latest stable version 2026」を検索
2. Dockerfile の `rust:1.90` と最新stableの差分を確認
3. セキュリティ修正を含むリリースがあれば報告

## 6. サマリ表示

結果を以下のフォーマットで表示:

```
## scan-backend 結果

### Rust依存関係
- 脆弱性: X件 (critical: X, high: X, moderate: X, low: X)
- パッチ更新可能: X件
- マイナー更新可能: X件
- メジャー更新可能: X件
- 認証関連クレート: 問題なし / 要確認 (詳細)

### Docker base image
- ビルド: rust:1.90-bullseye → 最新: rust:X.XX
- ランタイム: debian:bullseye-slim
  - EOL: YYYY-MM-DD (残りXヶ月)
  - 重大CVE: あり/なし

### OpenSSL (libssl1.1)
- バージョン: 1.1.1
- EOL状況: (サポート終了/サポート中)
- 推奨: (OpenSSL 3.x / bookwormへの移行等)

### Rust toolchain
- Dockerfile: 1.90 → 最新stable: X.XX.X (差分: Xリリース)

### 要対応事項
- (対応が必要な項目を優先度順に列挙)
- なければ「scan-backend: All Clear」
```

## 7. Issue起票判定

以下のルールに従い **ユーザー確認なしに自動起票** する。起票されたIssueが人間への最初の連絡となる。

| 条件                                       | Priority   | ラベル            | フォーマット |
| ------------------------------------------ | ---------- | ----------------- | ------------ |
| Critical脆弱性（CVSS 9.0+）                | P0         | `troubleshooting` | 4F           |
| High脆弱性（CVSS 7.0-8.9）                 | P1         | `troubleshooting` | 4F           |
| 認証関連クレートの脆弱性（severity問わず） | P0に格上げ | `troubleshooting` | 4F           |
| Base Image EOL 6ヶ月以内                   | P1         | `planning`        | STAR         |
| libssl1.1 EOL済み                          | P1         | `planning`        | STAR         |
| Moderate 3件以上蓄積                       | P2         | `troubleshooting` | 4F           |
| Rust メジャーバージョン遅延                | P2         | `planning`        | STAR         |

自動起票フロー:

1. `gh issue list -R uechikohei/spot-diggz --search "KEYWORD" --state all --limit 10` で重複チェック
2. 重複がなければ `gh issue create` で起票 → `gh project item-add 2 --owner uechikohei --url ISSUE_URL` でProject追加
3. 重複がある場合は起票スキップし、サマリで既存Issue番号を報告
