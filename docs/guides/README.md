# SpotDiggz Guides

- Status: Current index

[DR-0019](../decisions/0019-api-client-boundary.md)で独自Web UIは新MVPの対象外としました。以下は主に移行前実装の操作・設定手順です。Cloud Runの実行手順は未整備であり、[運用計画](../operations/cloud-run-plan.md)を確認してください。[現在の公開先は未確認](../operations/service-status.md)であり、Production設定guideは公開先と既存設定を確認するまで実行しません。内部設計は[`../architecture.md`](../architecture.md)、外部から観測できる契約は[`../specifications/`](../specifications/README.md)を参照します。

## Index

| Guide | Audience | Related specification | Status |
| --- | --- | --- | --- |
| [`how-to-use.md`](how-to-use.md) | SpotDiggz利用者 | [`web-ui.md`](../specifications/web-ui.md) | 旧Web操作の参照。新MVP対象外 |
| [`github-oauth-setup.md`](github-oauth-setup.md) | Production owner | [`web-ui.md`](../specifications/web-ui.md) | 旧設定の参照。新構成へ流用しない |
| [`slack-setup.md`](slack-setup.md) | Slack App owner | [`chat-integrations.md`](../specifications/chat-integrations.md) | 再設定参考。公開先未確認 |
| [`discord-setup.md`](discord-setup.md) | Discord App owner | [`chat-integrations.md`](../specifications/chat-integrations.md) | 再設定参考。公開先・Production E2E未確認 |

## Safety

- Token、client secret、signing secret、API key、database URLを文書、chat、command argument、shell historyへ貼りません。
- Secret入力はguideで指定した非表示promptまたはprovider secret storeを使います。
- External設定変更とProduction deployはownerの明示承認後に行います。
- GuideのProduction確認状況とlocal実装状況を混同しません。
