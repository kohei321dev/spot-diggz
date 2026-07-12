# Observability, SLI and SLO Design

- Status: Draft
- Date: 2026-07-12
- Scope: application logs, metrics, traces, AI evaluation, alerting

## 1. 目的

利用者が「今日どこで滑るか」を決める主要ジャーニーを、技術的な成功だけでなく、データ鮮度と推薦品質を含めて観測できるようにする。

観測性は次の問いへ答えるために設計する。

- 利用者は推薦を完了できたか
- 推薦は検証済み施設データに基づいていたか
- どの処理または外部依存が遅かったか
- AIは適切な候補を説明し、根拠のない情報を追加しなかったか
- デプロイ後に品質、レイテンシー、エラー、費用が変化したか
- 障害時に、利用者影響と原因候補を相関できるか

## 2. テレメトリーの使い分け

| Signal | 主な用途 | 適するデータ | 避けること |
|---|---|---|---|
| Log | 個別イベントの診断、監査、経緯 | エラーコード、結果、状態遷移、相関ID | 自由文のみ、秘密情報、全入力本文 |
| Metric | 集計、傾向、SLO、アラート | 件数、比率、分位値、キュー長、費用 | ユーザーID等の高カーディナリティラベル |
| Trace | 1要求の依存関係と遅延分析 | HTTP、DB、外部API、AI、ツール呼び出し | プロンプト本文、正確な位置、個人情報 |
| Event/Feedback | プロダクト成果とAI品質 | 推薦完了、遷移、誤情報報告、評価 | 同意のない行動追跡 |

OpenTelemetry等のベンダー非依存な標準を優先候補とする。採用は技術選定ADRで決定する。

## 3. 構造化ログ標準

### 共通フィールド

| Field | Required | Description |
|---|---:|---|
| `timestamp` | yes | UTC、ISO 8601 |
| `level` | yes | `debug` / `info` / `warn` / `error` |
| `service` | yes | 発生元サービスまたはモジュール |
| `environment` | yes | `local` / `preview` / `production` 等 |
| `release` | yes | Git SHAまたは成果物バージョン |
| `event_name` | yes | 安定した機械可読イベント名 |
| `message` | yes | 人が読める短い説明 |
| `outcome` | yes | `success` / `failure` / `rejected` / `degraded` |
| `request_id` | request | 1要求の相関ID |
| `trace_id` | request | 分散トレースとの相関ID |
| `session_hash` | optional | ローテーション可能な仮名識別子 |
| `duration_ms` | operation | 処理時間 |
| `error_code` | failure | 安定したエラー分類 |
| `retryable` | failure | 再試行可否 |

### イベント命名

`<domain>.<entity>.<past-tense-event>` の形式を基本とする。

例:

- `recommendation.request.received`
- `recommendation.candidates.filtered`
- `recommendation.response.completed`
- `recommendation.response.failed`
- `facility.catalog.loaded`
- `facility.source.marked_stale`
- `ai.explanation.completed`
- `ai.explanation.rejected`
- `navigation.external.opened`
- `feedback.accuracy.submitted`
- `deployment.released`

イベント名は文言変更の影響を受けない識別子として扱う。

### ログへ記録しない情報

- パスワード、APIキー、アクセストークン、Cookie、認証ヘッダー
- 正確な緯度経度、完全な住所、移動履歴
- メールアドレス、氏名、電話番号等の個人情報
- 利用者の自由入力全文
- AIのプロンプト全文、応答全文、モデルの内部推論
- 決済情報や秘密設定

診断上必要な場合は、分類、長さ、ハッシュ、テンプレートID、施設ID等の最小メタデータへ変換する。

### ログレベル

- `debug`: ローカルまたは短期診断用。productionでは既定無効またはサンプリングする。
- `info`: 正常な重要状態遷移、デプロイ、主要ジャーニー完了。
- `warn`: 縮退動作、鮮度期限超過、再試行成功、利用者影響前の異常。
- `error`: 利用者要求の失敗、データ破損、再試行枯渇、セキュリティ拒否。

例外が発生しただけで機械的にerrorにせず、最終的な利用者影響と回復可否で決める。

## 4. メトリクス設計

### 技術メトリクス

REDを基本にする。

- Rate: リクエスト数、推薦数、AI呼び出し数
- Errors: エラー数、拒否数、縮退数、タイムアウト数
- Duration: API、DB、外部API、AI、推薦全体のp50/p95/p99

追加候補:

- DB接続・クエリ時間
- 外部API成功率・レイテンシー・リトライ数
- キュー長、処理遅延、ジョブ失敗数（導入した場合）
- キャッシュヒット率（導入した場合）
- プロセス資源使用量

### プロダクトメトリクス

- 推薦入力開始数
- 有効入力完了率
- 推薦結果表示率
- 外部ナビゲーション遷移率
- 誤情報報告率
- 推薦後フィードバック率
- 再訪率

これらを成功の代理指標として扱い、利用者が実際に滑れたかという目的と混同しない。

### 施設データ品質メトリクス

- 出典URL保有率
- `verified_at`保有率
- 鮮度期限超過率
- 必須項目欠落率
- 更新ジョブ成功率
- 誤情報報告から修正までの時間

### AI品質・費用メトリクス

- `model_provider`、`model_name`、`model_version`
- `prompt_template_version`
- `evaluation_set_version`
- 入出力トークン数
- 推定費用
- モデル・ツール別レイテンシー
- ツール選択正解率、引数妥当率、実行成功率
- 構造化出力スキーマ成功率
- 根拠付与率
- 根拠のない施設事実の検出率
- フォールバック率、再生成率、拒否率
- 人間評価または利用者評価

モデル名、テンプレート版等は制御された低カーディナリティ値に限定する。

## 5. トレース設計

主要要求に次のspanを設定する。

```text
recommendation.request
  input.validate
  facility.query
  recommendation.filter
  recommendation.score
  ai.explanation.generate
    retrieval.query
    model.invoke
    output.validate
  response.serialize
```

各spanには、結果、所要時間、エラーコード、再試行回数、件数を記録する。
入力本文、施設の完全な住所、緯度経度、プロンプト、応答全文は記録しない。

## 6. SLI候補

SLIはユーザー視点の良いイベントと全イベントの比率、またはユーザーが経験する時間として定義する。

### Recommendation availability

```text
good = 有効な要求に対し、候補または「条件に合う候補なし」を正しく返した要求
total = 検証を通過した全推薦要求
```

利用者入力エラー、レート制限による意図的拒否は別分類とし、無条件に分母から除外しない。

### Recommendation latency

有効な要求の受付から、推薦候補と根拠を表示可能になるまでの時間を測る。
決定的推薦とAI説明を分けて測定し、AI障害時も基本推薦を返せるか確認する。

### Grounded response quality

施設に関する事実を含む推薦のうち、検証済み施設データと出典に対応しているものの割合。

### Facility freshness

推薦対象施設のうち、定義した鮮度期限内に検証され、必須項目が揃っている施設の割合。

### User journey completion

推薦入力を開始したセッションのうち、推薦結果を表示できたセッションの割合。

## 7. 仮SLOとSLA方針

外部顧客との補償契約がないMVP段階ではSLAを設定しない。
SLOは内部の意思決定目標として使用する。

初期リリース判定用の仮目標:

| SLI | Provisional objective | Window |
|---|---:|---|
| Recommendation availability | 99.0%以上 | rolling 30 days |
| Deterministic recommendation latency | p95 500ms以下 | rolling 7 days |
| AI explanation latency | p95 8秒以下 | rolling 7 days |
| Grounded facility facts | 100% | release evaluation set |
| Facility records with source and verified time | 100% | current catalog |

これらは契約値ではない。実利用データを2〜4週間収集し、利用者期待、費用、実装負荷を踏まえてADRで更新する。

## 8. エラーバジェット

availability SLOが99.0%の場合、30日間の悪いイベント許容率は1.0%となる。

運用方針案:

- バジェットが十分: 小さな機能実験を継続
- 50%以上消費: 新規リリースのリスクを下げ、原因上位を改善
- 80%以上消費: 信頼性修正を優先し、変更範囲を制限
- 使い切り: 非必須リリースを停止し、復旧・再発防止を優先

トラフィックが少ない期間は比率が不安定になるため、イベント数と絶対件数も併記する。

## 9. アラート設計

アラートは、利用者に症状があり、担当者が行動できる場合だけ通知する。

優先候補:

- 推薦availabilityの急速なエラーバジェット消費
- 推薦p95レイテンシーの継続的悪化
- AI失敗により決定的推薦も返せない状態
- 施設カタログ読み込み失敗
- 認証・認可拒否の異常増加
- 秘密情報漏えいまたはプロンプトインジェクション検知
- デプロイ直後のエラー率・品質低下

CPU、メモリ等は診断用に収集するが、それ単独で緊急通知しない。
単一閾値ではなく、短時間と長時間のバーンレートを組み合わせる。

各アラートには次を含める。

- 利用者影響
- 対象SLIと現在値
- 開始時刻と関連デプロイ
- ダッシュボードとトレースへのリンク
- 最初に確認する項目
- 縮退、ロールバック、無効化手順

## 10. ダッシュボード

### Product health

- 推薦開始、完了、離脱、外部遷移
- 誤情報報告と修正時間
- 施設データ鮮度

### Service health

- availability、レイテンシー、エラー率
- 外部依存別の成功率とレイテンシー
- デプロイマーカー

### AI quality and cost

- 評価スコア、根拠付与率、拒否率
- モデル・プロンプト版別の品質
- トークン、費用、レイテンシー
- ツール失敗とフォールバック

### Security

- 認証・認可拒否
- 入力検証・ガードレール拒否
- レート制限
- 秘密情報・PII検知

## 11. 保持・サンプリング案

| Data | Initial retention | Notes |
|---|---:|---|
| Application logs | 30日 | 個人情報を含めない |
| Security audit logs | 90日 | 改ざん防止とアクセス制限 |
| Metrics | 90日 | 長期は集約値へ間引く |
| Traces | 7日 | エラーは長め、正常系はサンプリング |
| AI evaluation results | 180日 | 入出力本文ではなく評価結果と版を保存 |

法的要件、費用、利用実態が判明した時点で更新する。保持期間終了後の削除を自動化する。

## 12. 実装時の受入条件

- 主要要求に相関IDがある
- 共通ログスキーマのテストがある
- 禁止情報がログへ出ないことをテストしている
- 主要SLIをダッシュボードで確認できる
- デプロイ版と障害を相関できる
- AI出力の構造検証と評価がある
- 主要アラートに行動手順がある
- 本番の失敗を回帰テストへ追加する手順がある
