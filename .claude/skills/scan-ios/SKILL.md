---
name: scan-ios
description: iOSレイヤーのセキュリティスキャン（Swift Package, Firebase iOS SDK, GoogleSignIn）
allowed-tools: Bash(gh issue:*), Bash(gh project:*)
---

iOSレイヤーのセキュリティ・バージョンスキャンを実行してください。

現在のSwift Package依存関係:
!`cat iOS/spot-diggz.xcodeproj/project.xcworkspace/xcshareddata/swiftpm/Package.resolved 2>/dev/null || echo "Package.resolved not found"`

手順:

## 1. Swift Package依存関係の確認

1. 上記で取得した `Package.resolved` の内容を解析
2. 各パッケージの現在バージョン（ピン留めバージョン）を抽出
3. 主要パッケージを一覧表示

## 2. Firebase iOS SDK の確認

1. `Package.resolved` からFirebase iOS SDKの現在バージョンを確認
2. WebSearch で「Firebase iOS SDK latest version release notes 2026」を検索
3. WebSearch で「Firebase iOS SDK CVE vulnerability 2025 2026」を検索
4. 現在バージョンと最新バージョンの差分を算出
5. 認証関連（Firebase Auth）のセキュリティ修正がないか重点確認

## 3. GoogleSignIn-iOS の確認

1. `Package.resolved` からGoogleSignIn-iOSの現在バージョンを確認
2. WebSearch で「GoogleSignIn-iOS latest version 2026」を検索
3. WebSearch で「GoogleSignIn-iOS CVE vulnerability 2025 2026」を検索
4. OAuth関連のセキュリティ修正がないか重点確認

## 4. AppAuth-iOS の確認

1. `Package.resolved` からAppAuth-iOSの現在バージョンを確認（GoogleSignInの推移依存）
2. WebSearch で「AppAuth-iOS CVE vulnerability 2025 2026」を検索
3. OAuth 2.0フロー関連のセキュリティ修正がないか確認

## 5. gRPC / その他の依存関係確認

1. `Package.resolved` からgRPC-swiftの現在バージョンを確認
2. WebSearch で「grpc-swift CVE vulnerability 2025 2026」を検索
3. その他の依存パッケージ（abseil, BoringSSL, leveldb, nanopb, Promises, GTMSessionFetcher等）について:
   - WebSearch で重大CVE（CVSS 7.0+）がないか一括確認
   - 特にBoringSSL（暗号化）は重点確認

## 6. Xcode・Swift ツールチェーン確認

1. WebSearch で「Xcode latest version 2026」を検索し最新バージョンを確認
2. WebSearch で「Swift latest version 2026」を検索し最新stableを確認
3. WebSearch で「Xcode CVE vulnerability 2025 2026」を検索
4. 最低デプロイターゲットのサポート状況を確認

## 7. サマリ表示

結果を以下のフォーマットで表示:

```
## scan-ios 結果

### Swift Package 依存関係
| パッケージ | 現在 | 最新 | CVE |
|-----------|------|------|-----|
| firebase-ios-sdk | X.X.X | X.X.X | なし/あり |
| GoogleSignIn-iOS | X.X.X | X.X.X | なし/あり |
| AppAuth-iOS | X.X.X | X.X.X | なし/あり |
| grpc-swift | X.X.X | X.X.X | なし/あり |
| (その他主要パッケージ) | ... | ... | ... |

### 認証関連
- Firebase Auth: 問題なし / 要確認 (詳細)
- GoogleSignIn: 問題なし / 要確認 (詳細)
- AppAuth (OAuth): 問題なし / 要確認 (詳細)

### ツールチェーン
- Xcode: 最新 X.X (使用中: X.X)
- Swift: 最新 X.X (使用中: X.X)

### 要対応事項
- (対応が必要な項目を優先度順に列挙)
- なければ「scan-ios: All Clear」
```

## 8. Issue起票判定

以下のルールに従い **ユーザー確認なしに自動起票** する。起票されたIssueが人間への最初の連絡となる。

| 条件                                                   | Priority   | ラベル            | フォーマット |
| ------------------------------------------------------ | ---------- | ----------------- | ------------ |
| Critical脆弱性（CVSS 9.0+）                            | P0         | `troubleshooting` | 4F           |
| High脆弱性（CVSS 7.0-8.9）                             | P1         | `troubleshooting` | 4F           |
| Firebase Auth / GoogleSignInの脆弱性（severity問わず） | P0に格上げ | `troubleshooting` | 4F           |
| メジャーバージョン遅延（Firebase SDK等）               | P2         | `planning`        | STAR         |
| Xcode最低バージョン要件変更                            | P1         | `planning`        | STAR         |

自動起票フロー:

1. `gh issue list -R uechikohei/spot-diggz --search "KEYWORD" --state all --limit 10` で重複チェック
2. 重複がなければ `gh issue create` で起票 → `gh project item-add 2 --owner uechikohei --url ISSUE_URL` でProject追加
3. 重複がある場合は起票スキップし、サマリで既存Issue番号を報告
