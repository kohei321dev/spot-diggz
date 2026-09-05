# UI Research Samples

- Status: Non-normative
- Canonical UI specification: [`../../specifications/web-ui.md`](../../specifications/web-ui.md)
- Current implementation: `internal/webui/static/`

このdirectoryのHTMLは、施設詳細、検索結果candidate、施設名linkだけの一覧を比較するために作成した静的sampleです。`おすすめ`等の検索条件依存文言は検索結果側のcontextで扱い、施設固有の恒久情報として固定しません。

HTMLのtitleや表示が現在のAccepted Decision Recordまたは実装と異なる場合、[`../../specifications/web-ui.md`](../../specifications/web-ui.md)と実装を優先します。特にYouTube iframeはADR-0014により、動画を含む施設詳細を利用者が開いた後だけ生成するのが現行仕様です。

## Samples

- [`facility-card-video-first-sample.html`](facility-card-video-first-sample.html): 検索結果candidate案
- [`facility-detail-video-first-sample.html`](facility-detail-video-first-sample.html): 施設詳細案
- [`facility-list-sample.html`](facility-list-sample.html): 施設名と詳細linkだけの一覧案
