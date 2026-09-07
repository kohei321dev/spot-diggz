# SpotDiggz Specifications

- Status: Current index

このディレクトリは、利用者、client、外部systemから観測できる現在の振る舞いを管理します。内部構成は[`../architecture.md`](../architecture.md)、判断理由は[`../decisions/README.md`](../decisions/README.md)を参照します。

## Index

| Specification | Related requirements | Status |
| --- | --- | --- |
| [`api-mvp.md`](api-mvp.md) | DR-0019/DR-0020による新MVPと既存要求IDの移行 | 方針Accepted、実装Incomplete |
| [`nearby-search.md`](nearby-search.md) | R-001〜R-003、R-018の入力と周辺検索への移行 | 入力方針Accepted、数値/応答/受信方式の詳細Incomplete、未実装 |
| [`web-ui.md`](web-ui.md) | R-001–R-005、R-007、R-013–R-017 | 旧Web実装の参照。新MVP対象外 |
| [`facility-data.md`](facility-data.md) | R-004、R-006、R-008–R-010、R-014–R-016、NFR-001–NFR-003 | Current |
| [`chat-integrations.md`](chat-integrations.md) | R-002–R-005、R-008、R-017–R-020 | 新返信方針と移行前実装を区別 |
| [`facility-catalog.openapi.yaml`](facility-catalog.openapi.yaml) | HTTP APIに関係する全要求 | 移行前実装の契約。新API認証は未反映 |

## Update rule

- 外部から観測できるUI、API、data format、error、provider縮退、chat responseを変更するPull Requestで、該当仕様を同時に更新します。
- OpenAPIと説明文が矛盾する場合は黙って解消せず、実装、test、Accepted Decision Record、Issueの根拠を再確認します。
- 未承認の調査・UI案は、本ディレクトリの仕様または実装を上書きしません。
