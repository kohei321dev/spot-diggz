# Continuous Delivery Pipeline Design

- Status: Draft
- Date: 2026-07-12
- Platform: 未選定

## 1. 目的

小さな変更を早く検証し、同じ手順で繰り返しリリースし、問題時に安全に戻せる状態を作る。
CI/CDはツール導入ではなく、Git上の変更を検証済み成果物へ変換する再現可能なプロセスとして設計する。

## 2. 原則

- Gitをコード、スキーマ、設定、パイプライン、インフラ定義のSingle Source of Truthにする。
- `main`を常にリリース可能に保つ。
- 長期ブランチを避け、小さな変更を頻繁に統合する。
- 失敗を早く返すため、安価で高速な検査を先に実行する。
- 成果物を一度だけビルドし、同一物を環境間で昇格させる。
- 手作業を前提にせず、同じ入力から同じ結果を再現できるようにする。
- デプロイ成功を完了とせず、スモークテストとSLI確認までをリリースに含める。
- ロールバックまたは安全な前方修正を事前に設計する。

## 3. ブランチとレビュー

- デフォルトブランチは `main`。
- 作業ブランチは短命にし、1つの目的に限定する。
- PRにはWHAT、WHY、受入条件、リスク、検証方法、ロールバック方法を記載する。
- 必須レビュー人数は個人開発では0または1とし、AIレビューを人間の承認の代替にしない。
- 必須チェックが成功するまでmainへ統合しない。
- mainへのforce pushと履歴書き換えを禁止する。

## 4. PRパイプライン

採用技術に応じて次の順序で構成する。

### Stage 1: Repository hygiene

目標時間: 1分以内。

- 変更ファイルと禁止ファイルの確認
- 改行、空白、生成物混入の検査
- Markdownリンクと設定構文の検査
- 秘密情報スキャン
- コミット・PRメタデータ検査

### Stage 2: Static verification

目標時間: 3分以内。

- formatter check
- linter
- type check
- API・設定・AI出力スキーマ検査
- 依存方向と循環依存の検査

### Stage 3: Automated tests

- ドメイン単体テスト
- コンポーネントテスト
- DB・外部境界の統合テスト
- 認証・認可・入力検証テスト
- 回帰テスト
- AI評価セット

テストは並列化するが、失敗原因が追跡できる粒度を保つ。

### Stage 4: Security and supply chain

- 依存関係脆弱性検査
- ライセンス検査
- SAST
- コンテナ・IaC検査（採用時）
- SBOM生成
- 依存関係とアクションのバージョン固定
- ビルド来歴と署名（本番公開時）

### Stage 5: Build

- production相当設定でビルド
- 成果物の再現性確認
- サイズと依存関係の予算検査
- Git SHA、ビルド日時、スキーマ版を成果物メタデータへ付与

## 5. mainパイプライン

```text
main update
  -> full verification
  -> build once
  -> artifact + SBOM + provenance
  -> preview/staging deploy
  -> smoke + migration compatibility
  -> production approval policy
  -> progressive deployment
  -> post-deploy SLI check
  -> promote or rollback
```

個人開発のMVPでは、常設staging環境を必須にしない。PR previewとproduction前スモークで要求を満たせるか、費用と運用負荷を比較してADRで決定する。

## 6. データベース変更

- スキーマ変更は後方互換なexpand/contractを基本とする。
- アプリとスキーマを同時に戻せない前提で設計する。
- migrationをバージョン管理し、適用済み状態を追跡する。
- 本番データを使わずにmigrationの前進・後退または安全な前方修正を検証する。
- 削除・型変更・必須化は、移行期間と利用状況確認後に行う。
- バックアップの存在だけでなく、復元テストを行う。

## 7. デプロイ戦略

初期候補:

- preview: PRごとの一時環境
- production: rollingまたはplatform標準の安全な置換
- 高リスク変更: canaryまたはfeature flag

feature flagには所有者、目的、既定値、削除期限を持たせる。恒久設定として放置しない。

デプロイ中は次を記録する。

- release SHA
- 開始・終了時刻
- 実行者または自動化主体
- DB migration版
- feature flag変更
- デプロイ方式と対象割合

## 8. リリース検証

デプロイ後に自動で確認する。

- health/readiness
- 主要ユーザージャーニーのスモークテスト
- APIエラー率とp95レイテンシー
- 推薦availability
- AI構造化出力・根拠付与率のサンプル
- 新しいセキュリティ拒否の異常増加
- データベースmigration状態

一定時間の観察窓を設け、基準を外れた場合は昇格を停止する。

## 9. ロールバック

リリースごとに次のいずれかを用意する。

- 直前成果物への即時切り戻し
- feature flagによる無効化
- AI機能から決定的推薦のみへの縮退
- 外部依存を無効化した縮退モード
- 後方互換を維持した前方修正

ロールバック条件を主観にせず、SLI、エラー率、セキュリティイベント、データ不整合で定義する。

## 10. CI/CD自体の観測

次を計測する。

- deployment frequency
- lead time for changes
- change failure rate
- failed deployment recovery time
- パイプライン成功率とp95所要時間
- flaky test率
- キュー待ち時間
- ロールバック回数
- セキュリティ修正までの時間

パイプラインの遅さや不安定さを放置すると、小さく頻繁な統合が妨げられるため、プロダクトと同様に改善対象とする。

## 11. 秘密情報と環境設定

- 秘密情報はGitへ保存しない。
- 環境差分は秘密ではない設定と秘密情報を分離する。
- CI/CDの権限は短命な認証情報と最小権限を使う。
- productionへの書き込み権限をPR検証ジョブへ与えない。
- third-party actionはcommit SHAまたは検証済み版へ固定する。
- fork PRや外部入力から秘密情報へアクセスさせない。

## 12. 実装開始時の決定事項

- CI/CDプラットフォーム
- preview/staging/productionの環境構成
- 成果物形式とregistry
- SBOM、署名、provenance方式
- DB migration方式
- feature flag方式
- rollbackと縮退方式
- 必須チェックと目標実行時間
- AI評価の合格条件

これらをADRで確定してから本番パイプラインを実装する。
