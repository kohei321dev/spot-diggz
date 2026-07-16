# ADR-0007 Go製モジュラーモノリスを初期アプリケーションruntimeに採用する

- Status: Accepted
- Date: 2026-07-16
- Related: [Product Baseline](../product_baseline.md)
- Related: [Quality Attributes](../architecture/quality-attributes.md)
- Related: [ADR-0006](0006-remove-legacy-implementation.md)

## Context

spot-diggzのMVPは、1都市圏の検証済み施設20〜30件を対象に、利用者の目的、レベル、時間、位置条件、交通手段から最大3件を推薦する。

初期トラフィックとデータ量は小さく、言語選定で高いthroughputを最優先する必要はない。一方、個人開発で継続できる運用負荷、施設情報と位置条件の安全な取り扱い、決定的推薦ロジックのテスト容易性、依存関係の脆弱性検査、再現可能なデプロイを重視する。

[ADR-0006](0006-remove-legacy-implementation.md)により、旧Go Metadata APIを含む旧実装は現行ツリーから削除した。本ADRは旧実装を復元する判断ではなく、現在のProduct Baselineと品質特性から新しいアプリケーションruntimeを選び直すものである。

## Decision

1. 初期アプリケーションruntimeはGoとする。
2. MVPは単一のデプロイ可能単位を基本とするモジュラーモノリスとして実装する。
3. Session Input、Facility Catalog、Deterministic Recommendation、Feedback、Observability、将来のAI Adapterを論理moduleとして分離するが、初期段階では別serviceへ分割しない。
4. 本番成果物は、1つのGo application binaryを含むOCI container imageとする。Databaseと外部providerは成果物へ含めない。
5. Web UIのframeworkと配信方式は別ADRで決定する。静的assetを採用する場合は、同一containerまたはGo binaryへの埋め込みを候補にできるが、本ADRでは確定しない。
6. API schema、外部providerの入出力、AI構造化出力は言語から独立したschemaとして管理し、境界で検証する。
7. CIでは最低限、format、静的検査、単体・component test、build、依存関係監査、Go sourceとbinaryの脆弱性検査、秘密情報検査を実行する。
8. Go toolchain、CI、container builderのversionを整合させ、security patchを継続的に取り込む。

## Rationale

Goを採用する主理由は高トラフィック性能ではなく、次の運用上の単純さである。

- applicationを単一binaryとしてbuildし、同じ成果物を各環境へ昇格できる
- production imageからcompiler、package manager、shell等を除外しやすい
- `gofmt`、`go vet`、`go test`、`go build`を標準toolchainで統一できる
- `govulncheck`でsourceの到達可能な脆弱性とbuild済みbinaryを検査できる
- 決定的推薦ロジックを静的型付けされたmoduleとしてテストしやすい
- runtime、rollback、障害調査の単位をapplication artifactへ揃えられる

言語だけでsecurityが保証されるわけではない。入力検証、認証・認可、秘密情報管理、位置情報の非保存、依存関係更新、container scan、監視を組み合わせる。

## Alternatives

### PythonとFastAPIを採用する

Pythonは、API schemaとvalidationを短く実装しやすく、将来のデータ分析、AI、地理処理で利用可能なlibraryが多い。開発者がPythonで大幅に速く反復できる場合は、有力な選択肢になる。

現在のMVPでは、施設の絞り込みと推薦を決定的な通常ロジックとして実装し、AIは後からadapterとして追加する。Python固有の処理は必須要件ではなく、旧実装を再利用せず新規に作る場合でも、interpreter、runtime package、application dependencyを含む成果物の管理が必要になる。

初期アプリケーション全体には採用しない。Python固有のlibraryが中核要件になる、または実測した開発速度と保守性でGoが継続的な障害になる場合に再検討する。

### TypeScriptとNode.jsでWeb UIとapplicationを統一する

型とpackage ecosystemをWeb UIと共有できる。一方、runtime dependencyとbuild成果物の境界が増え、単一binaryとGo向け脆弱性検査の利点を得られない。Web UIの技術選定が未確定な現時点では、application runtimeとして採用しない。

### 初期段階からserviceを分割する

Facility Catalog、Recommendation、AI Adapterを独立deployできるが、network障害、認証、observability、release、costの管理対象が増える。独立scale、障害分離、release cadenceの必要性が実測されていないため採用しない。

## Consequences

### Positive

- applicationのbuild、scan、deploy、rollback単位を1つにできる
- production artifactのruntime dependencyと攻撃面を小さくしやすい
- domain ruleと外部adapterの依存方向をcompileとtestで検証しやすい
- AI providerが停止しても決定的推薦を維持する構成にできる
- 将来service分割が必要になっても、先に論理moduleの境界を検証できる

### Negative

- 開発端末へversionを固定したGo toolchainが必要になる
- Python中心のAI・データ処理をapplication内へ直接導入しにくい
- API schema、validation、fixtureの記述量がPythonより増える可能性がある
- Go toolchain自体の脆弱性に対応するため、CIとcontainerを含めたpatch更新が必要になる
- Web UIを同じbinaryへ含める場合は、build順序とasset更新方法を別途設計する必要がある

## Verification

実装開始後、次をCIまたはrelease検証で確認する。

- 1つのcommandでapplication binaryを再現可能にbuildできる
- 同一commitから生成した同一container imageを環境間で昇格できる
- production containerがnon-rootで起動し、不要なruntime toolを含まない
- `go test`で主要domain rule、入力境界、失敗経路を検証できる
- `govulncheck`でsourceとbuild済みbinaryに許容不能な脆弱性がない
- container、dependency、secret scanがCIで成功する
- 位置条件と秘密情報がlogへ記録されないことをtestで確認できる
- module間に循環依存とdomainから外部SDKへの逆向き依存がない

## Revisit Conditions

次のいずれかを確認した場合、本ADRを再評価する。

- Python固有のAI、地理、データ処理libraryが主要user journeyに必須になる
- Goによる実装・保守時間がSprint Goal達成の継続的な障害になる
- 選定したhostingがGo artifactを合理的なcostと運用負荷で実行できない
- componentごとに独立したscale、障害分離、release cadenceが必要になる
- 単一deployable unitではSLOまたはsecurity boundaryを満たせない
