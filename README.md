# spot-diggz

スケートスポット検索・シェアアプリケーション（旧SkateSpotSearchのリプレイス）

## 現行プロダクト方針

現在は、スポットを地図に集約すること自体ではなく、利用者の気分・目的・レベル・時間・交通手段に応じて、
「今日滑る場所」を決めるための多言語Webサービスを検証する。

要求の基準は [docs/product_baseline.md](docs/product_baseline.md)、開発の進め方は
[docs/development_workflow.md](docs/development_workflow.md)、設計判断は [docs/adr/](docs/adr/) を参照する。
既存のiOS/Android/Rust実装は、方向性が検証されるまで削除せず保管する。

## Tech Stack

| Layer              | Technology                                |
| ------------------ | ----------------------------------------- |
| **Backend**        | Rust (スクラッチ実装)                     |
| **Frontend**       | React + TypeScript                        |
| **Mobile**         | iOS (Swift / SwiftUI)                     |
| **Infrastructure** | GCP (Cloud Run, Firestore, Cloud Storage) |
| **IaC**            | Terraform                                 |
| **CI/CD**          | GitHub Actions                            |

## Project Structure

```
spot-diggz/
├── web/
│   ├── api/               # Rust APIサーバー
│   ├── ui/                # React UIアプリ
│   ├── resources/         # Terraform Infrastructure
│   ├── scripts/           # 開発用スクリプト
│   └── sample/            # Seed用画像サンプル
├── iOS/                   # iOSアプリ
├── android/               # Androidアプリ（予定）
├── docs/                  # ドキュメント
├── .github/workflows/     # CI/CD
└── CLAUDE.md              # 開発・運用ルール
```

## 開発・運用

開発環境セットアップ、スラッシュコマンド、コーディング規約、運用ルール等の詳細は [AGENTS.md](AGENTS.md) と [CLAUDE.md](CLAUDE.md) を参照。

## 使うコマンド一覧

### Git

- `git status --short`: 作業ツリーの未コミット変更を確認する。
- `git branch archive/<snapshot-name>`: 方針転換や大きな整理の前に、現在の状態を退避する。
- `git switch -c product/<product-name>`: 新しいプロダクト方針を検証する統合ブランチを作成して切り替える。

今回の方針整理では、`archive/pre-session-planner-20260712` を退避ブランチ、
`product/session-planner` を作業ブランチとして使用した。

### 品質検証

- `cd web/ui && npm ci`: React UIの依存関係をlockfileどおりに再現する。
- `cd web/ui && npm run lint`: React UIのESLint検査を実行する。
- `cd web/ui && npm run type-check`: TypeScriptの型検査を実行する。
- `cd web/ui && npm test -- --coverage --watch=false`: React UIのユニットテストとカバレッジ計測を実行する。
- `cd web/ui && npm audit --audit-level=high`: npm依存関係の高severity以上の脆弱性を検査する。
- `cd web/ui && npm run build`: React UIの本番ビルドを検証する。
- `cd web/api && cargo fmt -- --check && cargo clippy -- -D warnings && cargo test --verbose`: Rust APIの整形、Lint、ユニットテストを実行する。
- `cd web/resources && terraform fmt -check -recursive && terraform init -backend=false && terraform validate`: Terraformの整形と設定検証を実行する。
