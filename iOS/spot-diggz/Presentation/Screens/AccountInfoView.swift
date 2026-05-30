import SwiftUI

/// Displays account metadata and the current user's own posts.
struct AccountInfoView: View {
    @EnvironmentObject var appState: SdzAppState

    let userId: String
    let displayName: String
    let email: String?
    let providerLabel: String
    let emailChangeNote: String
    let mySpots: [SdzSpot]

    var body: some View {
        ZStack {
            Color.sdzBackground.ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: SdzSpacing.lg + 2) {
                    accountCard
                    postCard
                }
                .padding(.horizontal, SdzSpacing.lg)
                .padding(.top, SdzSpacing.lg + 2)
                .padding(.bottom, SdzSpacing.xl)
            }
        }
        .navigationTitle("アカウント情報")
        .navigationBarTitleDisplayMode(.inline)
    }

    private var accountCard: some View {
        VStack(alignment: .leading, spacing: SdzSpacing.md) {
            HStack(spacing: SdzSpacing.md) {
                avatarView
                VStack(alignment: .leading, spacing: SdzSpacing.xxs) {
                    Text(displayName)
                        .font(SdzTypography.headline)
                        .foregroundColor(.sdzTextPrimary)
                    if let email, !email.isEmpty {
                        Text(email)
                            .font(SdzTypography.subheadline)
                            .foregroundColor(.sdzTextSecondary)
                    }
                }
                Spacer()
            }
            SdzDividerView()
            accountInfoRow(title: "ユーザーID", value: userId)
            accountInfoRow(title: "ログイン方法", value: providerLabel)
            if let email, !email.isEmpty {
                accountInfoRow(title: "メール", value: email)
            }
            Text(emailChangeNote)
                .font(SdzTypography.caption1)
                .foregroundColor(.sdzTextTertiary)
                .padding(.top, SdzSpacing.xs + 2)
        }
        .padding(SdzSpacing.lg)
        .background(Color.sdzSurfaceElevated)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous)
                .stroke(Color.sdzBorder, lineWidth: 1)
        )
    }

    private var postCard: some View {
        VStack(alignment: .leading, spacing: SdzSpacing.md) {
            Text("自分の投稿")
                .font(SdzTypography.headline)
                .foregroundColor(.sdzTextPrimary)
                .padding(.horizontal, SdzSpacing.lg)
                .padding(.top, SdzSpacing.lg)

            if mySpots.isEmpty {
                Text("投稿がありません")
                    .font(SdzTypography.subheadline)
                    .foregroundColor(.sdzTextSecondary)
                    .padding(.horizontal, SdzSpacing.lg)
                    .padding(.bottom, SdzSpacing.lg)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(mySpots.enumerated()), id: \.element.id) { index, spot in
                        NavigationLink(destination: SpotDetailView(spot: spot)) {
                            SpotCardView(spot: spot)
                                .padding(.horizontal, SdzSpacing.lg)
                                .padding(.vertical, SdzSpacing.xs)
                        }
                        .buttonStyle(.plain)
                        if index < mySpots.count - 1 {
                            SdzDividerView()
                                .padding(.horizontal, SdzSpacing.lg)
                        }
                    }
                }
                .padding(.bottom, SdzSpacing.sm)
            }
        }
        .background(Color.sdzSurfaceElevated)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous)
                .stroke(Color.sdzBorder, lineWidth: 1)
        )
    }

    private var avatarView: some View {
        Group {
            if let data = appState.profileImageData,
               let image = UIImage(data: data) {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
            } else {
                Circle()
                    .fill(Color.sdzBgTertiary)
                    .overlay(
                        Image(systemName: "person.fill")
                            .foregroundColor(.sdzTextSecondary)
                    )
            }
        }
        .frame(width: 56, height: 56)
        .clipShape(Circle())
    }

    private func accountInfoRow(title: String, value: String) -> some View {
        HStack {
            Text(title)
                .font(SdzTypography.subheadline)
                .foregroundColor(.sdzTextSecondary)
            Spacer(minLength: 10)
            Text(value)
                .font(SdzTypography.subheadline)
                .foregroundColor(.sdzTextPrimary)
                .lineLimit(1)
        }
    }
}

#if DEBUG
struct AccountInfoView_Previews: PreviewProvider {
    static var previews: some View {
        NavigationView {
            AccountInfoView(
                userId: "user-123",
                displayName: "Kohei",
                email: "kohei@example.com",
                providerLabel: "Google",
                emailChangeNote: "Googleログインのメール変更はアカウント側で行います。",
                mySpots: [SdzSpot.sample(id: "1", name: "サンプルスポット")]
            )
            .environmentObject(SdzAppState())
        }
    }
}
#endif
