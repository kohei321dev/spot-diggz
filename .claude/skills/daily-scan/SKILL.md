---
name: daily-scan
description: 日次総合セキュリティスキャン（全レイヤーを順次スキャンし統合サマリを出力）
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(gh pr:*), Bash(gh run:*), Bash(cargo audit:*), Bash(cargo outdated:*), Bash(npm audit:*), Bash(npm outdated:*), Bash(gh issue:*), Bash(gh project:*)
---

日次総合セキュリティスキャンを実行してください。全6フェーズを順に実行し、最後に統合サマリを表示します。

各フェーズの結果を蓄積し、Phase 5 で統合サマリとして一覧化してください。

現在のインフラ設定:

- Terraform バージョン: !`cat web/.terraform-version`
- Google provider: !`grep -A2 'google' web/resources/versions.tf 2>/dev/null || grep -r 'required_providers' web/resources/ --include='*.tf' -A5 2>/dev/null | head -20`

現在のiOS依存関係:
!`cat iOS/spot-diggz.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved 2>/dev/null || echo "Package.resolved not found"`

---

## Phase 0: リポジトリ状態の確認

1. `git status` でワーキングツリーの状態を確認
2. `gh pr list -R uechikohei/spot-diggz` で未マージPRを確認
3. `gh run list -R uechikohei/spot-diggz --limit 5` で直近のCI結果を確認

---

## Phase 1: インフラスキャン（/scan-infra 相当）

### 1-1. Terraform バージョン・CVE

1. 現在のTerraformバージョンを確認
2. WebSearch で Terraform最新安定版を確認
3. WebSearch で Terraform直近CVEを確認

### 1-2. Google Provider バージョン・CVE

1. 現在のGoogle providerバージョンを確認
2. WebSearch で最新版を確認
3. WebSearch で直近CVEを確認

### 1-3. GCPサービス脆弱性

1. WebSearch で Cloud Run / Firestore / Cloud Storage の直近脆弱性を確認

### 1-4. GitHub Actions

1. `.github/workflows/ci.yml` の使用アクション一覧を確認
2. `@master` 参照や古いバージョンを警告
3. WebSearch で GitHub Actions サプライチェーン攻撃情報を確認

### 1-5. tfsec

1. WebSearch で tfsec最新版を確認（現在 v1.0.3 使用中）

---

## Phase 2: バックエンドスキャン（/scan-backend 相当）

### 2-1. Rust依存関係

```
cd web/api
cargo audit
cargo outdated
```

1. severity（critical/high/moderate/low）で分類

### 2-2. 認証・セキュリティ関連クレート重点確認

1. WebSearch で jsonwebtoken / reqwest / base64 / serde の直近CVEを確認

### 2-3. Docker base image

1. WebSearch で `rust:1.90-bullseye` / `debian:bullseye-slim` の脆弱性を確認
2. Debian BullseyeのEOL状況を確認

### 2-4. libssl1.1 / OpenSSL

1. WebSearch で libssl1.1 / OpenSSL 1.1.1 の脆弱性・EOL状況を確認

### 2-5. Rust toolchain

1. WebSearch で Rust最新stableバージョンを確認（Dockerfile: rust:1.90）

---

## Phase 3: フロントエンドスキャン（/scan-frontend 相当）

### 3-1. npm依存関係

```
cd web/ui
npm audit
npm outdated
```

1. severity（critical/high/moderate/low）で分類

### 3-2. 重要パッケージ個別確認

1. WebSearch で firebase / react / vite / react-router-dom / leaflet の直近CVEを確認
2. Firebase JS SDK は認証コアのため重点確認

### 3-3. Node.js ランタイム

1. WebSearch で Node.js 20 LTS最新版・EOL日を確認

### 3-4. ビルドツール

1. WebSearch で TypeScript / ESLint v8 EOL / Vite のCVE情報を確認

### 3-5. npmサプライチェーン

1. WebSearch で npm サプライチェーン攻撃の最新情報を確認

---

## Phase 4: iOSスキャン（/scan-ios 相当）

### 4-1. Swift Package依存関係

1. 上記の `Package.resolved` を解析し各パッケージのバージョンを抽出

### 4-2. Firebase iOS SDK

1. WebSearch で Firebase iOS SDK最新版・CVEを確認
2. Firebase Auth のセキュリティ修正を重点確認

### 4-3. GoogleSignIn-iOS / AppAuth-iOS

1. WebSearch で GoogleSignIn-iOS / AppAuth-iOS の最新版・CVEを確認
2. OAuth関連のセキュリティ修正を重点確認

### 4-4. gRPC / その他

1. WebSearch で grpc-swift / BoringSSL の重大CVEを確認

### 4-5. Xcode・Swift

1. WebSearch で Xcode / Swift の最新バージョン・CVEを確認

---

## Phase 5: 統合サマリ表示

全結果を以下のフォーマットで統合サマリ表示:

```
# Daily Security Scan - YYYY/MM/DD

## リポジトリ状態
- ブランチ: (現在のブランチ)
- 未コミット変更: あり/なし
- 未マージPR: X件
- 直近CI: PASS/FAIL

## インフラ (Phase 1)
- Terraform: 現在 X.X.X → 最新 X.X.X | CVE: なし/あり
- Google Provider: 現在 ~> X.X → 最新 X.X.X | CVE: なし/あり
- GCPサービス: 問題なし / 要確認
- GitHub Actions: 問題なし / 要更新 (詳細)
- tfsec: 現在 v1.0.3 → 最新 X.X.X

## バックエンド (Phase 2)
- Rust脆弱性: X件 (C:X H:X M:X L:X)
- 認証関連クレート: 問題なし / 要確認
- Docker base: bullseye-slim EOL YYYY-MM-DD
- libssl1.1: EOL状況
- Rust toolchain: 1.90 → 最新 X.XX

## フロントエンド (Phase 3)
- npm脆弱性: X件 (C:X H:X M:X L:X)
- Firebase JS SDK: 現在 X.X.X → 最新 X.X.X
- Node.js 20 LTS: EOL YYYY-MM-DD
- サプライチェーン: 問題なし / 要確認

## iOS (Phase 4)
- Firebase iOS SDK: 現在 X.X.X → 最新 X.X.X | CVE: なし/あり
- GoogleSignIn: 現在 X.X.X → 最新 X.X.X | CVE: なし/あり
- gRPC/BoringSSL: 問題なし / 要確認
- Xcode/Swift: 最新バージョン情報

## 要対応事項（優先度順）
### P0（即時対応）
- (あれば列挙)

### P1（今週中に対応）
- (あれば列挙)

### P2（バックログ）
- (あれば列挙)

### All Clear
- (全て問題なければ)「本日の要対応事項はありません」
```

## Issue起票判定

統合サマリの要対応事項に基づき、以下のルールで **ユーザー確認なしに自動起票** する。
日次スキャンは「調査→重複チェック→起票→Project追加」まで一切止まらずに完遂すること。
起票されたIssueが人間への最初の連絡となり、対応要否は人間が判断する。

| 条件                                                     | Priority   | ラベル            | フォーマット |
| -------------------------------------------------------- | ---------- | ----------------- | ------------ |
| Critical脆弱性（CVSS 9.0+）                              | P0         | `troubleshooting` | 4F           |
| High脆弱性（CVSS 7.0-8.9）                               | P1         | `troubleshooting` | 4F           |
| 認証関連パッケージの脆弱性（全レイヤー、severity問わず） | P0に格上げ | `troubleshooting` | 4F           |
| Base Image / Node.js LTS EOL間近                         | P1         | `planning`        | STAR         |
| Moderate 3件以上蓄積（レイヤー横断で合算）               | P2         | `troubleshooting` | 4F           |
| メジャーバージョン遅延                                   | P2         | `planning`        | STAR         |
| サプライチェーン攻撃で関連パッケージ影響                 | P0         | `troubleshooting` | 4F           |
| GitHub Actions `@master` 参照                            | P1         | `troubleshooting` | 4F           |
| CI失敗                                                   | P1         | `troubleshooting` | 4F           |

自動起票フロー:

1. `gh issue list -R uechikohei/spot-diggz --search "KEYWORD" --state all --limit 10` で重複チェック
2. 重複がなければ `gh issue create` で起票
3. `gh project item-add 2 --owner uechikohei --url ISSUE_URL` でProject追加
4. 重複がある場合は起票をスキップし、サマリで既存Issue番号を報告
5. 全Issue起票完了後、統合サマリの末尾に起票結果一覧を追記
