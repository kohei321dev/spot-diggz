import SwiftUI
import UIKit

/// Displays settings and profile-related actions for the current user.
struct ProfileView: View {
    @EnvironmentObject var appState: SdzAppState
    @Environment(SdzThemeManager.self) var themeManager

    @State private var user: SdzUser?
    @State private var mySpots: [SdzSpot] = []
    @State private var errorMessage: String?
    @State private var showAvatarPicker: Bool = false
    @State private var avatarImages: [UIImage] = []
    @State private var showShareSheet: Bool = false
    @State private var showAboutSheet: Bool = false
    @State private var showFeedbackAlert: Bool = false
    @State private var feedbackAlertMessage: String = ""

    var body: some View {
        Group {
            if #available(iOS 16.0, *) {
                NavigationStack {
                    profileContent
                }
            } else {
                NavigationView {
                    profileContent
                }
            }
        }
        .onAppear {
            loadData()
        }
        .sheet(isPresented: $showAvatarPicker) {
            SdzImagePicker(maxSelection: 1) { newImages in
                avatarImages = newImages
            }
        }
        .sheet(isPresented: $showShareSheet) {
            SdzShareSheet(activityItems: [shareMessage])
        }
        .sheet(isPresented: $showAboutSheet) {
            aboutSheet
        }
        .alert("ご意見フォームを開けません", isPresented: $showFeedbackAlert) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(feedbackAlertMessage)
        }
        .onChange(of: avatarImages) { _, newValue in
            guard let image = newValue.last else {
                return
            }
            if let data = image.jpegData(compressionQuality: 0.85) {
                appState.setProfileImageData(data)
            } else if let data = image.pngData() {
                appState.setProfileImageData(data)
            }
        }
    }

    private var profileContent: some View {
        ZStack {
            Color.sdzBackground.ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: SdzSpacing.lg + 2) {
                    if let errorMessage {
                        Text(errorMessage)
                            .font(SdzTypography.caption1)
                            .foregroundColor(.sdzError)
                            .padding(.horizontal, SdzSpacing.md)
                    }
                    sectionLabel("設定")
                    settingsCard
                    sectionLabel("テーマ")
                    themeCard
                    sectionLabel("アプリ")
                    appCard
                }
                .padding(.horizontal, SdzSpacing.lg)
                .padding(.top, SdzSpacing.md)
                .padding(.bottom, SdzSpacing.xl)
            }
        }
        .navigationTitle("設定")
        .navigationBarTitleDisplayMode(.inline)
    }

    private var settingsCard: some View {
        VStack(spacing: 0) {
            NavigationLink {
                AccountInfoView(
                    userId: resolvedUserId,
                    displayName: resolvedDisplayName,
                    email: resolvedEmail,
                    providerLabel: providerLabel,
                    emailChangeNote: emailChangeNote,
                    mySpots: mySpots
                )
                .environmentObject(appState)
            } label: {
                SettingsRowView(iconName: "person.crop.circle", title: "アカウント情報")
            }
            SdzDividerView()
            NavigationLink {
                MyListDetailView()
                    .environmentObject(appState)
            } label: {
                SettingsRowView(
                    iconName: "heart.circle",
                    title: "マイリスト",
                    value: "\(appState.favoriteSpots.count)"
                )
            }
            SdzDividerView()
            Button {
                showAvatarPicker = true
            } label: {
                SettingsRowView(iconName: "camera.circle", title: "アイコン変更")
            }
            .buttonStyle(.plain)
            SdzDividerView()
            Button {
                appState.signOut()
            } label: {
                SettingsRowView(
                    iconName: "rectangle.portrait.and.arrow.right",
                    title: "ログアウト",
                    showsChevron: false,
                    isDestructive: true
                )
            }
            .buttonStyle(.plain)
        }
        .background(Color.sdzSurfaceElevated)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous)
                .stroke(Color.sdzBorder, lineWidth: 1)
        )
    }

    private var themeCard: some View {
        VStack(spacing: 0) {
            HStack(spacing: SdzSpacing.md) {
                ForEach(SdzColorSchemePreference.allCases) { preference in
                    Button {
                        withAnimation(SdzAnimation.standard) {
                            themeManager.preference = preference
                        }
                    } label: {
                        VStack(spacing: SdzSpacing.xs) {
                            Image(systemName: preference.iconName)
                                .font(SdzTypography.title3)
                                .foregroundColor(
                                    themeManager.preference == preference
                                        ? .sdzStreet
                                        : .sdzTextTertiary
                                )
                            Text(preference.label)
                                .font(SdzTypography.caption1)
                                .foregroundColor(
                                    themeManager.preference == preference
                                        ? .sdzTextPrimary
                                        : .sdzTextTertiary
                                )
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, SdzSpacing.md)
                        .background(
                            themeManager.preference == preference
                                ? Color.sdzStreet.opacity(0.1)
                                : Color.clear
                        )
                        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.sm, style: .continuous))
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(SdzSpacing.lg)
        }
        .background(Color.sdzSurfaceElevated)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous)
                .stroke(Color.sdzBorder, lineWidth: 1)
        )
    }

    private var appCard: some View {
        VStack(spacing: 0) {
            Button {
                showShareSheet = true
            } label: {
                SettingsRowView(iconName: "square.and.arrow.up", title: "シェア")
            }
            .buttonStyle(.plain)
            SdzDividerView()
            Button {
                openFeedbackMail()
            } label: {
                SettingsRowView(iconName: "paperplane", title: "ご意見・ご要望")
            }
            .buttonStyle(.plain)
            SdzDividerView()
            Button {
                showAboutSheet = true
            } label: {
                SettingsRowView(iconName: "sparkles", title: "SpotDiggzについて")
            }
            .buttonStyle(.plain)
        }
        .background(Color.sdzSurfaceElevated)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous)
                .stroke(Color.sdzBorder, lineWidth: 1)
        )
    }

    private func sectionLabel(_ title: String) -> some View {
        Text(title)
            .font(SdzTypography.subheadlineMedium)
            .foregroundColor(.sdzTextSecondary)
            .padding(.horizontal, SdzSpacing.xxs)
    }

    private var resolvedUserId: String {
        user?.userId ?? appState.authUserId ?? "-"
    }

    private var resolvedDisplayName: String {
        if let user, !user.displayName.isEmpty, user.displayName != "unknown" {
            return user.displayName
        }
        if let displayName = appState.authDisplayName, !displayName.isEmpty {
            return displayName
        }
        if let email = appState.authEmail, !email.isEmpty {
            return email
        }
        return "ユーザー"
    }

    private var resolvedEmail: String? {
        if let user, let email = user.email, !email.isEmpty {
            return email
        }
        return appState.authEmail
    }

    private var providerLabel: String {
        let providers = Set(appState.authProviderIds)
        var labels: [String] = []
        if providers.contains("google.com") {
            labels.append("Google")
        }
        if providers.contains("apple.com") {
            labels.append("Apple")
        }
        if providers.contains("password") {
            labels.append("メール/パスワード")
        }
        return labels.isEmpty ? "不明" : labels.joined(separator: " / ")
    }

    private var emailChangeNote: String {
        if appState.authProviderIds.contains("google.com") || appState.authProviderIds.contains("apple.com") {
            return "Google/Appleログインのメール変更は各アカウント側で行います。アプリ内での変更には未対応です。"
        }
        return "メール変更は今後対応予定です。"
    }

    private var shareMessage: String {
        "SpotDiggzでスケートスポットを探そう。"
    }

    private var appVersionText: String {
        let version = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "-"
        let build = Bundle.main.object(forInfoDictionaryKey: "CFBundleVersion") as? String ?? "-"
        return "バージョン \(version) (\(build))"
    }

    private var aboutSheet: some View {
        VStack(spacing: SdzSpacing.lg) {
            Capsule()
                .fill(Color.sdzTextTertiary)
                .frame(width: 42, height: 5)
                .padding(.top, SdzSpacing.sm + 2)
            Text("SpotDiggzについて")
                .font(SdzTypography.headline)
            Text(appVersionText)
                .font(SdzTypography.subheadline)
                .foregroundColor(.sdzTextSecondary)
            Text("スケートスポットを地図と一覧で見つけ、投稿できるアプリです。")
                .font(SdzTypography.caption1)
                .multilineTextAlignment(.center)
                .foregroundColor(.sdzTextSecondary)
                .padding(.horizontal, SdzSpacing.xl - 4)
            Button("閉じる") {
                showAboutSheet = false
            }
            .buttonStyle(.borderedProminent)
            .padding(.bottom, SdzSpacing.md)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .background(Color.sdzBackground.ignoresSafeArea())
        .presentationDetents([.fraction(0.35), .medium])
    }

    private func openFeedbackMail() {
        guard let encodedSubject = "SpotDiggz ご意見・ご要望".addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
              let url = URL(string: "mailto:?subject=\(encodedSubject)") else {
            feedbackAlertMessage = "メール起動リンクの生成に失敗しました。"
            showFeedbackAlert = true
            return
        }
        guard UIApplication.shared.canOpenURL(url) else {
            feedbackAlertMessage = "メールアプリが利用できません。"
            showFeedbackAlert = true
            return
        }
        UIApplication.shared.open(url)
    }

    private func loadData() {
        errorMessage = nil
        guard appState.idToken != nil else {
            errorMessage = "ログインが必要です。"
            return
        }
        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)

        Task {
            async let fetchedSpots = apiClient.fetchSpots(includeAuth: true)
            async let fetchedUser = apiClient.fetchCurrentUser()

            let spotsResponse: Result<[SdzSpot], Error>
            do {
                spotsResponse = .success(try await fetchedSpots)
            } catch {
                spotsResponse = .failure(error)
            }

            let userResponse: Result<SdzUser, Error>
            do {
                userResponse = .success(try await fetchedUser)
            } catch {
                userResponse = .failure(error)
            }

            await MainActor.run {
                switch userResponse {
                case .success(let currentUser):
                    user = currentUser
                case .failure(let error):
                    if let message = resolveErrorMessage(error) {
                        errorMessage = message
                    }
                }

                switch spotsResponse {
                case .success(let spots):
                    let currentUserId = user?.userId ?? appState.authUserId
                    if let currentUserId {
                        mySpots = spots.filter { $0.userId == currentUserId }
                    } else {
                        mySpots = []
                    }
                case .failure(let error):
                    if let message = resolveErrorMessage(error) {
                        errorMessage = message
                    }
                    mySpots = []
                }
            }
        }
    }

    private func resolveErrorMessage(_ error: Error) -> String? {
        if let apiError = error as? SdzApiError {
            switch apiError {
            case let .api(statusCode, _) where statusCode == 401 || statusCode == 404:
                return nil
            case let .statusCode(statusCode) where statusCode == 401 || statusCode == 404:
                return nil
            case .authRequired:
                return "ログインが必要です。"
            default:
                return apiError.errorDescription ?? "読み込みに失敗しました。"
            }
        }
        return (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
    }
}

private struct SdzShareSheet: UIViewControllerRepresentable {
    let activityItems: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: activityItems, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

#if DEBUG
struct ProfileView_Previews: PreviewProvider {
    static var previews: some View {
        ProfileView()
            .environmentObject(SdzAppState())
            .environment(SdzThemeManager())
    }
}
#endif
