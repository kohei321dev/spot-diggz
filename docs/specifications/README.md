# SpotDiggz Specifications

- Status: Current index

このディレクトリは、利用者、client、外部systemから観測できる現在の振る舞いを管理します。内部構成は[`../architecture.md`](../architecture.md)、判断理由は[`../decisions/README.md`](../decisions/README.md)を参照します。

## Index

| Specification | Related requirements | Status |
| --- | --- | --- |
| [`web-ui.md`](web-ui.md) | R-001–R-005、R-007、R-013–R-017 | Current |
| [`facility-data.md`](facility-data.md) | R-004、R-006、R-008–R-010、R-014–R-016、NFR-001–NFR-003 | Current |
| [`chat-integrations.md`](chat-integrations.md) | R-002–R-005、R-008、R-017–R-020 | Current |
| [`facility-catalog.openapi.yaml`](facility-catalog.openapi.yaml) | HTTP APIに関係する全要求 | Current contract |

## Update rule

- 外部から観測できるUI、API、data format、error、provider縮退、chat responseを変更するPull Requestで、該当仕様を同時に更新します。
- OpenAPIと説明文が矛盾する場合は黙って解消せず、実装、test、Accepted Decision Record、Issueの根拠を再確認します。
- ResearchにあるHTMLは案であり、本ディレクトリの仕様または実装を上書きしません。
