# ADR-002: SpotDiggz の repository boundary を API + Browser と Mobile Clients に分離する

## Status

Accepted

## Context

[事実] ADR-001では、SpotDiggz API本体をGo製の Skate Spot Metadata API として再設計する判断をAcceptedにした。

[事実] 2026-05-30の整理前は、Go API、Browser UI、legacy runtime、旧deployment証跡、mobile途中実装が混在していた。

[事実] 今回の設計入力では、このrepositoryをAPIとBrowser UIで構成し、iOSとAndroidは完全に別project / 別repositoryへ分離する方針が示された。

[推測] API/Browser と iOS/Android は、release cadence、CI、署名、store配布、secret管理、review観点が異なるため、active development repositoryを分ける方が保守しやすい。

[推測] Browser UIはAPI契約、OpenAPI、Vercel/Neon環境、debug/admin用途と密接に連動するため、初期段階ではAPI repository内に置く方が変更追従が速い。

## Decision

SpotDiggz本repositoryは、Go製 Skate Spot Metadata API と、そのAPIを利用するBrowser UIのrepositoryとして育てる。

iOS app と Android app は、本repositoryのactive source treeから外し、それぞれ別project / 別repositoryで管理する。

既存のlegacy runtime、旧deployment証跡、iOS途中実装、Android設計メモや途中成果物は、active source treeには残さず、archive branchに退避する。

Browser UIは、API contractと同時に育てる間は本repositoryに残す。将来、Browser UIがAPIと独立したproduct release cadenceを持つ場合は、別repository化を再検討する。

## Target Repository Layout

```text
kohei321dev/spot-diggz
  Go API
  OpenAPI
  PostgreSQL schema / migrations
  Browser UI
  API / Browser CI/CD

kohei321dev/spotdiggz-ios
  iOS app
  Swift / SwiftUI
  Xcode / TestFlight / App Store release workflow

kohei321dev/spotdiggz-android
  Android app
  Kotlin / Compose
  Android Studio / Play Console release workflow
```

## In Scope For This Repository

- Go API implementation under `cmd/` and `internal/`
- PostgreSQL schema and migrations under `db/`
- OpenAPI contract under `docs/openapi.yaml`
- API architecture and ADR docs
- Browser UI
- API/Browser local development and CI/CD
- Vercel/Neon oriented deployment configuration when introduced

## Out Of Scope For This Repository

- iOS app active source code
- Android app active source code
- Xcode project and Xcode Cloud workflow
- Gradle/Android build configuration
- App Store / TestFlight release pipeline
- Play Console release pipeline
- legacy API / deployment active CI/CD

## Rationale

[事実] ADR-001の主目的は、SpotDiggzを地図アプリ全体ではなく Skate Spot Metadata API として再設計すること。

[推測] Mobile appを同一repositoryに残すと、API変更・Browser変更のPRにXcode/Gradle/project file差分、署名設定、store配布設定が混ざりやすくなる。

[推測] APIとBrowser UIを同一repositoryに置くと、OpenAPI変更、local smoke test、preview環境、admin/debug UIの更新を同じPRで扱いやすい。

[推測] Mobile clientsはOpenAPI contractを外部契約として参照し、必要に応じてgenerated clientや手書きclientを各repositoryで管理する方が責務が明確になる。

## Consequences

- 本repositoryのactive source treeから `iOS/` と将来の `android/` active app codeを外す。
- legacy runtime、旧deployment証跡、mobile途中成果物はarchive branchで参照可能にする。
- CI/CDはGo APIとBrowser UIを主対象に整理する。
- Mobile clientsは別repositoryからAPI contractを参照する。
- API contractのbreaking changeは、Browser UIだけでなくmobile repositoriesへの影響も明示して扱う。

## Migration Plan

1. legacy runtime、旧deployment証跡、iOS/Android途中成果物を含むarchive branchを作成または更新する。
2. 本repositoryからlegacy runtime、旧deployment証跡、mobile active source treeを削除するcleanup branchを作る。
3. Go API、PostgreSQL、OpenAPI、Browser UI中心のREADME / CI / docsへ整理する。
4. iOS repositoryとAndroid repositoryを作る場合は、archive branchから必要なコードだけを移送する。
5. Mobile repositoriesでは、本repositoryのOpenAPIを契約として参照する。

## Open Questions

[未検証] iOS / Android repository名を `spotdiggz-ios` / `spotdiggz-android` にするか。

[未検証] Browser UIを `web/ui` のまま残すか、`apps/browser` へ再配置するか。

[未検証] OpenAPI contractをmobile repositoriesへどう配布するか。候補はGitHub release artifact、package、submodule、手動同期。

[未検証] Vercel上でAPIとBrowser UIを1 Projectにまとめるか、Root Directoryを分けた複数Projectにするか。
