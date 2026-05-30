# ユーザー識別/ユーザー名ポリシー

## 目的

- 認証方式が異なっても、同一ユーザーを安定して識別できる状態を作る
- 表示名やメール変更の影響範囲を明確にする

## 公式識別子 (Canonical ID)

- **Firebase Auth UID を唯一の識別子とする**
- API/Firestore では `userId = UID` を前提に参照・更新する
- すべてのデータリレーションは `userId` で管理する

## 表示名 (displayName)

- **任意入力**（未設定可）
- **重複は許容**（現時点は一意性を要件にしない）
- UI表示の優先順位: `displayName` → `email` → `ユーザー`
- 一意性が必要になった場合は `userHandle` を別フィールドで追加し、重複チェックを行う

## メールアドレス

- メール/パスワード認証は **アプリ内で変更可能**（将来対応）
- Google/Apple SSO は **アプリ内で変更不可**（各アカウント側で変更）
- メールは連絡用属性であり、識別子としては扱わない

## Firestore (users/{uid})

- 保持フィールド:
  - `userId` (UID)
  - `displayName` (任意)
  - `email` (任意)
  - `createdAt`, `updatedAt`
- `displayName` 未設定の場合は空/未設定で保存し、UI側でフォールバック表示する

## 影響範囲

- Tier 1 マスターデータ: `userId` は管理者のみが使用（バッチ投入時の createdBy）
- Tier 2 ユーザー個人データ: SwiftData + CloudKit で管理。`userId` でフィルタリング
- マイリスト: SwiftData + CloudKit で管理（Firestore API 経由から移行済み）
- アカウント削除時: SwiftData ローカルストアと CloudKit データを `userId` で特定して削除
- 表示名やメール変更はUI表示のみ影響し、データの紐付けには影響しない

## アーキテクチャ変更（2026-02-28）

- ユーザー個人データ（Tier 2 スポット、マイリスト）は Firestore に保存しない
- Firestore の `users/{uid}` ドキュメントは認証・プロファイル用途のみ継続
- 詳細: `docs/designs/tier2-spot-data-architecture.md`
