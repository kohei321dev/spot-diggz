import SwiftUI
import MapKit

/// A screen displaying detailed information about a specific spot.
struct SpotDetailView: View {
    @EnvironmentObject var appState: SdzAppState
    @State private var spot: SdzSpot

    @State private var approvalMessage: String?
    @State private var isRequestingApproval: Bool = false
    @State private var showNavigationDialog: Bool = false
    @State private var showImageViewer: Bool = false
    @State private var selectedImageIndex: Int = 0

    init(spot: SdzSpot) {
        _spot = State(initialValue: spot)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if !spot.images.isEmpty {
                    TabView(selection: $selectedImageIndex) {
                        ForEach(spot.images.indices, id: \.self) { index in
                            let urlString = spot.images[index]
                            Group {
                                if let url = URL(string: urlString) {
                                    AsyncImage(url: url) { phase in
                                        switch phase {
                                        case .empty:
                                            ProgressView()
                                                .frame(maxWidth: .infinity, maxHeight: .infinity)
                                        case .success(let image):
                                            image
                                                .resizable()
                                                .scaledToFill()
                                        case .failure:
                                            Color.gray.opacity(0.2)
                                        @unknown default:
                                            Color.gray.opacity(0.2)
                                        }
                                    }
                                    .contentShape(Rectangle())
                                    .onTapGesture {
                                        selectedImageIndex = index
                                        showImageViewer = true
                                    }
                                } else {
                                    Color.gray.opacity(0.2)
                                }
                            }
                            .tag(index)
                        }
                    }
                    .frame(height: 220)
                    .tabViewStyle(.page)
                    .clipShape(RoundedRectangle(cornerRadius: SdzRadius.md, style: .continuous))
                }

                VStack(alignment: .leading, spacing: 6) {
                    Text(spot.name)
                        .font(.title2)
                        .bold()
                    HStack(spacing: 8) {
                        Text("公開ステータス: \(approvalStatusText)")
                            .font(.caption)
                            .foregroundColor(approvalStatusColor)
                        if !spot.tags.isEmpty {
                            Text(spot.tags.joined(separator: ", "))
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                    }
                }

                if let description = spot.description, !description.isEmpty {
                    Text(description)
                        .font(.body)
                }

                locationSection

                if let parkAttributes = spot.parkAttributes {
                    parkDetailSection(parkAttributes)
                }
                let streetAttributes = spot.streetAttributes ?? SdzStreetAttributes(
                    surfaceMaterial: nil,
                    surfaceCondition: nil,
                    sections: nil,
                    difficulty: nil,
                    notes: nil
                )
                let streetTitle = spot.parkAttributes == nil ? "ストリート情報" : "路面・セクション"
                streetDetailSection(streetAttributes, title: streetTitle)

                metadataSection

                actionSection

                if isOwnedByCurrentUser {
                    ownerSection
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
        }
        .navigationTitle("スポット詳細")
        .navigationBarTitleDisplayMode(.inline)
        .fullScreenCover(isPresented: $showImageViewer) {
            SdzImageGalleryView(
                imageUrls: spot.images,
                initialIndex: selectedImageIndex
            )
        }
        .confirmationDialog("ナビを開く", isPresented: $showNavigationDialog, titleVisibility: .visible) {
            Button("Google Mapsで開く") {
                openGoogleMaps()
            }
            Button("Apple Mapsで開く") {
                openAppleMaps()
            }
        }
        .onAppear {
            refreshSpotDetail()
        }
    }

    private var locationSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("位置情報")
                .font(.headline)
            if let location = spot.location {
                Text("緯度: \(String(format: "%.5f", location.lat))  経度: \(String(format: "%.5f", location.lng))")
                    .font(.caption)
                    .foregroundColor(.secondary)
            } else {
                Text("未設定")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
    }

    private var metadataSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("その他")
                .font(.headline)
            Text("画像: \(spot.images.count) 枚")
                .font(.caption)
                .foregroundColor(.secondary)
            Text("作成日: \(formatDate(spot.createdAt))")
                .font(.caption)
                .foregroundColor(.secondary)
            Text("更新日: \(formatDate(spot.updatedAt))")
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }

    private var actionSection: some View {
        VStack(spacing: 8) {
            Button(action: {
                Task {
                    await appState.toggleFavorite(spot)
                }
            }) {
                Label(
                    isFavorite ? "お気に入り解除" : "お気に入りに追加",
                    systemImage: isFavorite ? "heart.fill" : "heart"
                )
                .frame(maxWidth: .infinity)
            }
            Button(action: {
                showNavigationDialog = true
            }) {
                Label("ナビ", systemImage: "car")
                    .frame(maxWidth: .infinity)
            }
            .disabled(spot.location == nil)
        }
        .buttonStyle(.bordered)
    }

    private var ownerSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            if let approvalMessage = approvalMessage {
                Text(approvalMessage)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if canRequestApproval {
                Button(isRequestingApproval ? "申請中..." : "承認を申請") {
                    requestApproval()
                }
                .buttonStyle(.borderedProminent)
                .disabled(isRequestingApproval)
            }
            NavigationLink("投稿を編集") {
                EditSpotView(spot: spot) { updated in
                    self.spot = updated
                }
            }
        }
        .padding(.top, 4)
    }

    private var isFavorite: Bool {
        appState.isFavorite(spot)
    }

    private var isOwnedByCurrentUser: Bool {
        guard let currentUserId = appState.authUserId else {
            return false
        }
        return spot.userId == currentUserId
    }

    private var approvalStatusText: String {
        switch spot.approvalStatus {
        case .approved:
            return "承認済"
        case .pending:
            return "審査中"
        case .rejected:
            return "差戻し"
        case .none:
            return "未申請"
        }
    }

    private var approvalStatusColor: Color {
        switch spot.approvalStatus {
        case .approved:
            return .sdzSuccess
        case .pending:
            return .sdzWarning
        case .rejected:
            return .sdzError
        case .none:
            return .sdzTextTertiary
        }
    }

    private var canRequestApproval: Bool {
        guard isOwnedByCurrentUser else {
            return false
        }
        switch spot.approvalStatus {
        case .none, .rejected:
            return true
        default:
            return false
        }
    }

    @ViewBuilder
    private func parkDetailSection(_ attributes: SdzSpotParkAttributes) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("スケートパーク情報")
                .font(.headline)
            if let hours = attributes.businessHours {
                VStack(alignment: .leading, spacing: 4) {
                    Text("営業時間")
                        .font(.subheadline)
                    ForEach(businessHoursLines(hours), id: \.self) { line in
                        Text(line)
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }
            }
            if let accessInfo = attributes.accessInfo, !accessInfo.isEmpty {
                Text("アクセス: \(accessInfo)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if let phone = attributes.phoneNumber, !phone.isEmpty {
                if let url = URL(string: "tel://\(phone)") {
                    Link("電話: \(phone)", destination: url)
                        .font(.caption)
                } else {
                    Text("電話: \(phone)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }
            if let officialUrl = attributes.officialUrl,
               !officialUrl.isEmpty,
               let url = URL(string: officialUrl) {
                Link("公式サイトを開く", destination: url)
                    .font(.caption)
            } else {
                Text("公式サイト: 入力されていません")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .padding(.top, 12)
    }

    @ViewBuilder
    private func streetDetailSection(_ attributes: SdzStreetAttributes, title: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
            if let material = attributes.surfaceMaterial, !material.isEmpty {
                Text("路面: \(material)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            } else {
                Text("路面: 入力されていません")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if let condition = attributes.surfaceCondition {
                if let roughness = condition.roughness, !roughness.isEmpty {
                    Text("粗さ: \(roughness)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                } else {
                    Text("粗さ: 入力されていません")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                if let crack = condition.crack, !crack.isEmpty {
                    Text("ひび割れ: \(crack)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                } else {
                    Text("ひび割れ: 入力されていません")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            } else {
                Text("粗さ: 入力されていません")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text("ひび割れ: 入力されていません")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if let difficulty = attributes.difficulty, !difficulty.isEmpty {
                Text("難易度: \(difficulty)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            } else {
                Text("難易度: 入力されていません")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            ratingLegend
            if let notes = attributes.notes, !notes.isEmpty {
                Text("備考: \(notes)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            } else {
                Text("備考: 補足などテキストが入力可能です")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            if let sections = attributes.sections, !sections.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text("セクション")
                        .font(.subheadline)
                    ForEach(Array(sections.enumerated()), id: \.offset) { _, section in
                        VStack(alignment: .leading, spacing: 2) {
                            Text(section.type)
                                .font(.caption)
                                .bold()
                            Text(sectionSummary(section))
                                .font(.caption2)
                                .foregroundColor(.secondary)
                        }
                    }
                }
            }
        }
        .padding(.top, 12)
    }

    private func businessHoursLines(_ hours: SdzSpotBusinessHours) -> [String] {
        let scheduleType = hours.scheduleType ?? .regular
        var lines: [String] = []

        switch scheduleType {
        case .regular:
            if hours.is24Hours {
                lines.append("24時間営業")
            } else {
                if let weekday = hours.weekday {
                    lines.append("平日 \(formatTimeRange(weekday))")
                }
                if hours.sameAsWeekday {
                    lines.append("週末: 平日と同じ")
                } else if let weekend = hours.weekend {
                    lines.append("週末 \(formatTimeRange(weekend))")
                }
            }
        case .weekdayOnly:
            if hours.is24Hours {
                lines.append("平日のみ 24時間営業")
            } else if let weekday = hours.weekday {
                lines.append("平日のみ \(formatTimeRange(weekday))")
            }
        case .weekendOnly:
            if hours.is24Hours {
                lines.append("週末のみ 24時間営業")
            } else if let weekend = hours.weekend {
                lines.append("週末のみ \(formatTimeRange(weekend))")
            }
        case .irregular:
            lines.append("不定休")
        case .schoolOnly:
            lines.append("スクールのみ")
        case .manual:
            lines.append("手動入力")
        }

        if let note = hours.note, !note.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            lines.append("備考: \(note)")
        }

        return lines
    }

    private func formatTimeRange(_ range: SdzSpotTimeRange) -> String {
        "\(formatMinutes(range.startMinutes)) - \(formatMinutes(range.endMinutes))"
    }

    private func formatMinutes(_ minutes: Int) -> String {
        let hour = minutes / 60
        let minute = minutes % 60
        return String(format: "%02d:%02d", hour, minute)
    }

    private func sectionSummary(_ section: SdzStreetSection) -> String {
        var parts: [String] = []
        if let count = section.count {
            parts.append("\(count)基")
        }
        if let height = section.heightCm {
            parts.append("高さ\(height)cm")
        }
        if let width = section.widthCm {
            parts.append("幅\(width)cm")
        }
        if let notes = section.notes, !notes.isEmpty {
            parts.append(notes)
        }
        return parts.isEmpty ? "詳細なし" : parts.joined(separator: " / ")
    }

    private func requestApproval() {
        guard appState.idToken != nil else {
            approvalMessage = "ログインが必要です。"
            return
        }
        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
        isRequestingApproval = true
        approvalMessage = nil
        Task {
            do {
                let input = SdzUpdateSpotInput(
                    name: spot.name,
                    description: nil,
                    location: nil,
                    tags: nil,
                    images: nil,
                    approvalStatus: .pending,
                    parkAttributes: nil,
                    streetAttributes: nil
                )
                let updated = try await apiClient.updateSpot(id: spot.spotId, input: input)
                await MainActor.run {
                    self.spot = updated
                    self.isRequestingApproval = false
                    self.approvalMessage = "承認申請を送信しました。"
                }
            } catch {
                let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
                await MainActor.run {
                    self.isRequestingApproval = false
                    self.approvalMessage = message
                }
            }
        }
    }

    private var spotCoordinate: CLLocationCoordinate2D? {
        guard let location = spot.location else {
            return nil
        }
        return CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng)
    }

    private func openGoogleMaps() {
        guard let coordinate = spotCoordinate else {
            return
        }
        SdzMapNavigator.openGoogleMaps(destination: coordinate, mode: .drive)
    }

    private func openAppleMaps() {
        guard let coordinate = spotCoordinate else {
            return
        }
        SdzMapNavigator.openAppleMaps(destination: coordinate, mode: .drive)
    }

    private var ratingLegend: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text("粗さ: 1=スムーズ / 3=気にならない / 5=悪い")
            Text("ひび割れ: 1=ほぼない / 3=たまにある / 5=多い")
            Text("難易度: 1=初心者には難しい / 3=幅広く楽しめる / 5=上級者向け")
        }
        .font(.caption2)
        .foregroundColor(.secondary)
    }

    private func formatDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "ja_JP")
        formatter.dateFormat = "yyyy/MM/dd"
        return formatter.string(from: date)
    }

    private func refreshSpotDetail() {
        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
        Task {
            do {
                let updated = try await apiClient.fetchSpotDetail(
                    id: spot.spotId,
                    includeAuth: true
                )
                await MainActor.run {
                    spot = updated
                    if spot.images.isEmpty {
                        selectedImageIndex = 0
                    } else {
                        selectedImageIndex = min(selectedImageIndex, spot.images.count - 1)
                    }
                }
            } catch {
                // Keep showing existing details on refresh failure.
            }
        }
    }
}

private struct SdzImageGalleryView: View {
    @Environment(\.dismiss) private var dismiss
    let imageUrls: [String]
    @State private var selection: Int

    init(imageUrls: [String], initialIndex: Int) {
        self.imageUrls = imageUrls
        _selection = State(initialValue: initialIndex)
    }

    var body: some View {
        ZStack(alignment: .topTrailing) {
            Color.black.ignoresSafeArea()
            TabView(selection: $selection) {
                ForEach(Array(imageUrls.enumerated()), id: \.offset) { index, urlString in
                    ZStack {
                        if let url = URL(string: urlString) {
                            AsyncImage(url: url) { phase in
                                switch phase {
                                case .empty:
                                    ProgressView()
                                case .success(let image):
                                    image
                                        .resizable()
                                        .scaledToFit()
                                case .failure:
                                    Color.gray.opacity(0.2)
                                @unknown default:
                                    Color.gray.opacity(0.2)
                                }
                            }
                        } else {
                            Color.gray.opacity(0.2)
                        }
                    }
                    .tag(index)
                }
            }
            .tabViewStyle(.page)

            Button(action: {
                dismiss()
            }) {
                Image(systemName: "xmark.circle.fill")
                    .font(.title)
                    .foregroundColor(.white)
                    .padding()
            }
        }
    }
}

#if DEBUG
struct SpotDetailView_Previews: PreviewProvider {
    static var previews: some View {
        NavigationView {
            SpotDetailView(spot: SdzSpot.sample(id: "preview", name: "プレビュー"))
                .environmentObject(SdzAppState())
        }
    }
}
#endif
