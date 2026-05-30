# リポジトリ構成方針

spot-diggz は web/ 配下に Web版（API/UI/IaC/運用スクリプト）を集約し、
モバイルは iOS/android で分離するモノレポ構成を採用する。

## 現状

- `web/` : Webアプリ一式（API/UI/IaC/スクリプト/seed資材）
- `iOS/` : iOSアプリ（準備中）
- `android/` : Androidアプリ（準備中）
- `docs/` : 設計/運用ドキュメント
- `AGENTS.md` : Codex向け運用ルール

```
spot-diggz/
  .devcontainer/
  .github/
  web/
    api/
    ui/
    resources/
    scripts/
    sample/
    sdz_seed_spots.sh
    firebase.json
    firestore.rules
    .firebaserc
    .terraform-version
  iOS/
  android/
  docs/
  AGENTS.md
  .gitignore
  README.md
  spot-diggz.code-workspace
```

## 補足
- Terraform / Firebase の設定ファイルは web/ に集約する。
- seed用の画像とスクリプトも web/ 配下にまとめる。
