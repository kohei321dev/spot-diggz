# SpotDiggz Guides

- Status: Current index

利用者またはownerが目的を達成するための現在の操作・設定手順です。内部設計は[`../architecture.md`](../architecture.md)、外部から観測できる契約は[`../specifications/`](../specifications/README.md)を参照します。

## Index

| Guide | Audience | Related specification | Status |
| --- | --- | --- | --- |
| [`how-to-use.md`](how-to-use.md) | SpotDiggz利用者 | [`web-ui.md`](../specifications/web-ui.md) | Current |
| [`github-oauth-setup.md`](github-oauth-setup.md) | Production owner | [`web-ui.md`](../specifications/web-ui.md) | Current。owner login確認済み |
| [`slack-setup.md`](slack-setup.md) | Slack App owner | [`chat-integrations.md`](../specifications/chat-integrations.md) | Current。owner flow確認済み |
| [`discord-setup.md`](discord-setup.md) | Discord App owner | [`chat-integrations.md`](../specifications/chat-integrations.md) | Current。Production E2EはIncomplete |

## Safety

- Token、client secret、signing secret、API key、database URLを文書、chat、command argument、shell historyへ貼りません。
- Secret入力はguideで指定した非表示promptまたはprovider secret storeを使います。
- External設定変更とProduction deployはownerの明示承認後に行います。
- GuideのProduction確認状況とlocal実装状況を混同しません。
