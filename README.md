# spot-diggz

スケートスポット検索・シェアアプリケーション（旧SkateSpotSearchのリプレイス）

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

開発環境セットアップ、スラッシュコマンド、コーディング規約、運用ルール等の詳細は [CLAUDE.md](CLAUDE.md) を参照。
