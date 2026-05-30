# UIトークン/ガイドライン（Web/iOS/Android）

## 目的
- Web/iOS/Android で同じ見た目と体験を維持するための共通トークンを定義する。
- 実装時の判断を減らし、UIの一貫性と保守性を上げる。

## カラーパレット（Web基準）
| Token | Value | 用途 |
| --- | --- | --- |
| sdz-bg | #f6f0e4 | 画面背景 |
| sdz-bg-soft | #fdf8f0 | 補助背景 |
| sdz-ink | #0e1b1f | 基本文字色 |
| sdz-muted | #5d6a73 | 補助文字色 |
| sdz-accent | #ff6b35 | プライマリアクション |
| sdz-accent-dark | #c94e1d | ホバー/押下 |
| sdz-surface | #ffffff | カード背景 |
| sdz-surface-alt | #f1f3f2 | サブカード背景 |
| sdz-border | #e1d9cc | 境界線 |

## タイポグラフィ
- 見出し: Space Grotesk / 600-700
- 本文: Noto Sans JP / 400-500
- 代替フォント: `sans-serif`

## 角丸（Radius）
- Pill: 999
- Card: 20
- Input: 14
- Chip: 999

## 余白スケール
- 4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 64

## シャドウ
- sdz-shadow: `0 24px 60px rgba(15, 18, 22, 0.12)`

## iOS適用指針（SwiftUI）
- 色: `Color("SdzAccent")` のように Asset Catalog に登録して参照する。
- フォント: `Font.custom("Space Grotesk", size:)` を見出しに使用する。
- 角丸: `RoundedRectangle(cornerRadius: 20)` をカードに適用する。
- 余白: 上記スケールに合わせて `padding()` を統一する。

## Android適用指針（Compose）
- 色: `Color(0xFF...)` を `SdzColors` として定義し共通利用する。
- フォント: `Typography` に Space Grotesk / Noto Sans JP を登録する。
- 角丸: `RoundedCornerShape(20.dp)` をカードに適用する。
- 余白: `Spacing` オブジェクトでスケールを共通化する。

## 参照
- `web/ui/src/index.css`
