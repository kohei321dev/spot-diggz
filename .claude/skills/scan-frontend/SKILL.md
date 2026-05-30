---
name: scan-frontend
description: フロントエンドレイヤーのセキュリティスキャン（npm依存関係, Node.js, React, Firebase JS SDK）
allowed-tools: Bash(npm audit:*), Bash(npm outdated:*), Bash(gh issue:*), Bash(gh project:*)
---

フロントエンドレイヤーのセキュリティ・バージョンスキャンを実行してください。

手順:

## 1. npm依存関係の監査

```
cd web/ui
npm audit
npm outdated
```

1. `npm audit` の結果からCVE一覧を抽出し、severity（critical/high/moderate/low）で分類
2. `npm outdated` の結果からパッチ・マイナー・メジャー更新可能なパッケージを分類

## 2. 重要パッケージの個別CVE確認

以下のパッケージについて、`npm audit` の結果に加えてWebSearchで個別に最新のCVE情報を確認:

| パッケージ                  | 用途                               | 重要度   |
| --------------------------- | ---------------------------------- | -------- |
| `firebase`                  | Firebase JS SDK（認証・Firestore） | 認証コア |
| `react` / `react-dom`       | UIフレームワーク                   | コア     |
| `react-router-dom`          | ルーティング                       | コア     |
| `vite`                      | ビルドツール                       | ビルド   |
| `leaflet` / `react-leaflet` | 地図表示                           | 機能     |
| `@mui/material`             | UIコンポーネント（使用時）         | UI       |

1. WebSearch で「{パッケージ名} npm CVE vulnerability 2025 2026」を検索（認証関連を優先）
2. 特に `firebase` JS SDKの脆弱性は認証コアのため重点確認
3. 脆弱性が見つかった場合、CVSS スコアと影響範囲を記録

## 3. Node.js ランタイム確認

1. WebSearch で「Node.js 20 LTS latest version security update 2026」を検索
2. CI（`.github/workflows/ci.yml`）で `node-version: '20'` を使用中 → 最新LTSパッチとの差分を確認
3. Node.js 20 LTS のEOL日を確認

## 4. TypeScript・ビルドツール確認

1. WebSearch で「TypeScript CVE vulnerability 2025 2026」を検索
2. WebSearch で「ESLint v8 EOL deprecation 2025 2026」を検索（v8 → v9移行状況）
3. WebSearch で「Vite CVE vulnerability 2025 2026」を検索

## 5. npmサプライチェーン攻撃情報

1. WebSearch で「npm supply chain attack 2025 2026」を検索
2. 直近で話題になったパッケージ乗っ取り・typosquatting等があれば報告
3. 本プロジェクトの依存パッケージに関連するものがないか確認

## 6. サマリ表示

結果を以下のフォーマットで表示:

```
## scan-frontend 結果

### npm依存関係
- 脆弱性: X件 (critical: X, high: X, moderate: X, low: X)
- パッチ更新可能: X件
- マイナー更新可能: X件
- メジャー更新可能: X件

### 重要パッケージ
- firebase: 現在 X.X.X → 最新 X.X.X (CVE: あり/なし)
- react: 現在 X.X.X → 最新 X.X.X
- vite: 現在 X.X.X → 最新 X.X.X (CVE: あり/なし)
- react-router-dom: 現在 X.X.X → 最新 X.X.X
- leaflet: 現在 X.X.X → 最新 X.X.X

### Node.js
- CI使用: 20.x → 最新LTS: X.X.X
- EOL: YYYY-MM-DD

### ビルドツール
- TypeScript: CVE なし/あり
- ESLint: v8 EOL状況
- Vite: CVE なし/あり

### サプライチェーン
- 直近の脅威情報: あり/なし (詳細)

### 要対応事項
- (対応が必要な項目を優先度順に列挙)
- なければ「scan-frontend: All Clear」
```

## 7. Issue起票判定

以下のルールに従い **ユーザー確認なしに自動起票** する。起票されたIssueが人間への最初の連絡となる。

| 条件                                      | Priority   | ラベル            | フォーマット |
| ----------------------------------------- | ---------- | ----------------- | ------------ |
| Critical脆弱性（CVSS 9.0+）               | P0         | `troubleshooting` | 4F           |
| High脆弱性（CVSS 7.0-8.9）                | P1         | `troubleshooting` | 4F           |
| Firebase JS SDKの脆弱性（severity問わず） | P0に格上げ | `troubleshooting` | 4F           |
| Moderate 3件以上蓄積                      | P2         | `troubleshooting` | 4F           |
| Node.js LTS EOL 6ヶ月以内                 | P1         | `planning`        | STAR         |
| メジャーバージョン遅延（React, Vite等）   | P2         | `planning`        | STAR         |
| サプライチェーン攻撃で関連パッケージ影響  | P0         | `troubleshooting` | 4F           |

自動起票フロー:

1. `gh issue list -R uechikohei/spot-diggz --search "KEYWORD" --state all --limit 10` で重複チェック
2. 重複がなければ `gh issue create` で起票 → `gh project item-add 2 --owner uechikohei --url ISSUE_URL` でProject追加
3. 重複がある場合は起票スキップし、サマリで既存Issue番号を報告
