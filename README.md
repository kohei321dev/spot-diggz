# spot-diggz

スケートスポット検索・シェアアプリケーション（旧SkateSpotSearchのリプレイス）

## Tech Stack

| Layer              | Technology                                |
| ------------------ | ----------------------------------------- |
| **Backend**        | Go (Skate Spot Metadata API)              |
| **Frontend**       | React + TypeScript                        |
| **Mobile**         | iOS (Swift / SwiftUI)                     |
| **Infrastructure** | Local Docker Compose / PostgreSQL（初期） |
| **IaC**            | Terraform                                 |
| **CI/CD**          | GitHub Actions                            |

## Project Structure

```
spot-diggz/
├── cmd/
│   └── api/               # Go APIエントリポイント
├── db/                    # PostgreSQL schema / goose migration
├── internal/              # Go API内部package
├── web/
│   ├── api/               # Rust legacy APIサーバー（移行完了まで保持）
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

## 使うコマンド一覧

| コマンド | 意味 |
| --- | --- |
| `go fmt ./...` | Go APIのformatを適用する |
| `go vet ./...` | Go APIの静的検査を実行する |
| `go test ./...` | Go APIのユニットテストを実行する |
| `go build -o bin/spotdiggz-api ./cmd/api` | Go APIの単一バイナリを生成する |
| `govulncheck ./...` | Go source/dependencyの脆弱性を確認する |
| `govulncheck -mode=binary ./bin/spotdiggz-api` | 生成済みGo binaryの脆弱性を確認する |

## ローカルPostgreSQL

初期実装では `DATABASE_URL` が未設定の場合は in-memory store、設定済みの場合は PostgreSQL store を使います。

```bash
docker compose up -d postgres
DATABASE_URL='postgres://spotdiggz:spotdiggz@localhost:5432/spotdiggz?sslmode=disable' go run ./cmd/api
```

停止する場合は `docker compose down` を使います。永続化volumeを消したい場合だけ `docker compose down -v` を使います。

ローカルDBの初期化SQLは [db/schema.sql](db/schema.sql) です。migration管理用に同等の goose migration を [db/migrations/00001_create_sdz_spots.sql](db/migrations/00001_create_sdz_spots.sql) に置いています。

## API設計メモ

- Go runtimeは `go.mod` の `go 1.25.10` を基準にする。
- ADR: [docs/adr/001-go-skate-spot-metadata-api.md](docs/adr/001-go-skate-spot-metadata-api.md)
- ADR: [docs/adr/002-repository-boundary-api-browser-mobile.md](docs/adr/002-repository-boundary-api-browser-mobile.md)
- 実装計画: [docs/go-api-implementation-plan.md](docs/go-api-implementation-plan.md)
- OpenAPI: [docs/openapi.yaml](docs/openapi.yaml)
