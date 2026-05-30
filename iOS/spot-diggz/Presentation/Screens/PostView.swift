import SwiftUI
import PhotosUI

/// A screen for creating a new skate spot.
struct PostView: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject var appState: SdzAppState
    @State private var name: String = ""
    @State private var description: String = ""
    @State private var tagsString: String = ""
    @State private var spotCategory: SdzSpotCategory = .street
    @State private var parkOfficialUrl: String = ""
    @State private var parkAccessInfo: String = ""
    @State private var parkPhoneNumber: String = ""
    @State private var parkScheduleType: SdzSpotBusinessScheduleType = .regular
    @State private var parkScheduleNote: String = ""
    @State private var parkIs24Hours: Bool = false
    @State private var parkSameAsWeekday: Bool = true
    @State private var parkWeekdayStart: Date = SdzTimeDefaults.weekdayStart
    @State private var parkWeekdayEnd: Date = SdzTimeDefaults.weekdayEnd
    @State private var parkWeekendStart: Date = SdzTimeDefaults.weekendStart
    @State private var parkWeekendEnd: Date = SdzTimeDefaults.weekendEnd
    @State private var streetSurfaceMaterialOption: String = "未入力"
    @State private var streetSurfaceMaterialCustom: String = ""
    @State private var streetRoughnessLevel: Int? = nil
    @State private var streetCrackLevel: Int? = nil
    @State private var streetDifficultyLevel: Int? = nil
    @State private var streetNotes: String = ""
    @State private var streetSections: [SdzStreetSectionInput] = []
    @State private var selectedLocation: SdzSpotLocation?
    @State private var imageItems: [SdzSpotImageItem] = []
    @State private var countryTag: String?
    @State private var lastResolvedCountryLocation: SdzSpotLocation?
    @State private var showImagePicker: Bool = false
    @State private var showUrlPickerSheet: Bool = false
    @State private var isSubmitting: Bool = false
    @State private var submitMessage: String?
    @State private var sharedUrlCandidate: String = ""
    @State private var showSharedUrlAlert: Bool = false
    @State private var sharedLocationCandidate: SdzSpotLocation?
    @State private var sharedLocationName: String = ""
    @State private var sharedLocationErrorMessage: String = ""
    @State private var showSharedLocationAlert: Bool = false
    @State private var showSharedLocationErrorAlert: Bool = false
    private let maxImagesPerSpot: Int = 3
    private let streetSurfaceOptions: [String] = ["未入力", "コンクリート", "アスファルト", "木製", "その他"]

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("スポット種別")) {
                    Picker("種類", selection: $spotCategory) {
                        ForEach(SdzSpotCategory.allCases) { category in
                            Text(category.label).tag(category)
                        }
                    }
                    .pickerStyle(.segmented)
                }

                Section(header: Text("基本情報")) {
                    TextField("スポット名", text: $name)
                    TextField("説明", text: $description, axis: .vertical)
                }

                Section(header: Text("タグ（カンマ区切り）")) {
                    TextField("例: パーク, ストリート", text: $tagsString)
                }

                Section(header: Text("位置情報")) {
                    if let selectedLocation {
                        Text("緯度: \(selectedLocation.lat)")
                        Text("経度: \(selectedLocation.lng)")
                            .foregroundColor(.secondary)
                    } else {
                        Text("スポット画面の地図でピンを選んでから投稿してください。")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                }

                Section(header: imageSectionHeader) {
                    SdzSpotImageRow(items: $imageItems, maxImages: maxImagesPerSpot) {
                        showImagePicker = true
                    }
                    Text("左端がメイン画像です。ドラッグで並び替えできます。")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                if spotCategory == .park {
                    Section(header: Text("スケートパーク情報")) {
                        HStack {
                            TextField("公式サイトURL", text: $parkOfficialUrl)
                                .textInputAutocapitalization(.never)
                                .keyboardType(.URL)
                            Button {
                                showUrlPickerSheet = true
                            } label: {
                                Image(systemName: "safari")
                            }
                            .buttonStyle(.bordered)
                        }
                        Picker("営業時間タイプ", selection: $parkScheduleType) {
                            ForEach(SdzSpotBusinessScheduleType.allCases) { scheduleType in
                                Text(scheduleType.label).tag(scheduleType)
                            }
                        }
                        .pickerStyle(.menu)

                        if parkScheduleType == .irregular || parkScheduleType == .schoolOnly || parkScheduleType == .manual {
                            TextField("営業時間の補足 (例: 不定休/スクール専用/手動入力)", text: $parkScheduleNote)
                            Text("詳細は公式サイトや現地案内をご確認ください。")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        } else {
                            Toggle("24時間営業", isOn: $parkIs24Hours)
                            if !parkIs24Hours {
                                switch parkScheduleType {
                                case .regular:
                                    DatePicker(
                                        "平日 開始",
                                        selection: $parkWeekdayStart,
                                        displayedComponents: .hourAndMinute
                                    )
                                    DatePicker(
                                        "平日 終了",
                                        selection: $parkWeekdayEnd,
                                        displayedComponents: .hourAndMinute
                                    )
                                    Toggle("週末も同じ", isOn: $parkSameAsWeekday)
                                    if !parkSameAsWeekday {
                                        DatePicker(
                                            "週末 開始",
                                            selection: $parkWeekendStart,
                                            displayedComponents: .hourAndMinute
                                        )
                                        DatePicker(
                                            "週末 終了",
                                            selection: $parkWeekendEnd,
                                            displayedComponents: .hourAndMinute
                                        )
                                    }
                                case .weekdayOnly:
                                    DatePicker(
                                        "平日 開始",
                                        selection: $parkWeekdayStart,
                                        displayedComponents: .hourAndMinute
                                    )
                                    DatePicker(
                                        "平日 終了",
                                        selection: $parkWeekdayEnd,
                                        displayedComponents: .hourAndMinute
                                    )
                                case .weekendOnly:
                                    DatePicker(
                                        "週末 開始",
                                        selection: $parkWeekendStart,
                                        displayedComponents: .hourAndMinute
                                    )
                                    DatePicker(
                                        "週末 終了",
                                        selection: $parkWeekendEnd,
                                        displayedComponents: .hourAndMinute
                                    )
                                case .irregular, .schoolOnly:
                                    EmptyView()
                                case .manual:
                                    EmptyView()
                                }
                            }
                        }
                        TextField("アクセス情報", text: $parkAccessInfo)
                        TextField("電話番号", text: $parkPhoneNumber)
                    }
                }

                Section(header: Text(spotCategory == .park ? "路面・セクション（任意）" : "ストリート情報")) {
                    Picker("路面", selection: $streetSurfaceMaterialOption) {
                        ForEach(streetSurfaceOptions, id: \.self) { option in
                            Text(option).tag(option)
                        }
                    }
                    .pickerStyle(.menu)
                    .onChange(of: streetSurfaceMaterialOption) { _, newValue in
                        if newValue != "その他" {
                            streetSurfaceMaterialCustom = ""
                        }
                    }
                    if streetSurfaceMaterialOption == "その他" {
                        TextField("路面の詳細を入力", text: $streetSurfaceMaterialCustom)
                    }
                    ratingPicker(
                        title: "粗さ",
                        selection: $streetRoughnessLevel,
                        helpText: "1: スムーズ / 3: 気にならない / 5: 悪い"
                    )
                    ratingPicker(
                        title: "ひび割れ",
                        selection: $streetCrackLevel,
                        helpText: "1: ほぼない / 3: たまにある / 5: 多い"
                    )
                    ratingPicker(
                        title: "難易度",
                        selection: $streetDifficultyLevel,
                        helpText: "1: 初心者には難しい / 3: 幅広く楽しめる / 5: 上級者向け"
                    )
                    TextField("備考（補足など）", text: $streetNotes, axis: .vertical)
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("セクション")
                                .font(.subheadline)
                            Spacer()
                            Button("追加") {
                                streetSections.append(SdzStreetSectionInput())
                            }
                        }
                        ForEach($streetSections) { $section in
                            SdzStreetSectionEditor(section: $section) {
                                streetSections.removeAll { $0.id == section.id }
                            }
                        }
                    }
                }

                Section {
                    if let submitMessage = submitMessage {
                        Text(submitMessage)
                            .foregroundColor(.secondary)
                    }
                    Button(isSubmitting ? "送信中..." : "投稿する") {
                        submit()
                    }
                    .disabled(isSubmitting)
                }
            }
            .navigationTitle("新しいスポットを投稿")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("閉じる") {
                        closeComposer()
                    }
                    .disabled(isSubmitting)
                }
            }
            .onAppear {
                appState.isPostingSpot = true
                applyDraftLocationIfNeeded()
                if countryTag == nil {
                    countryTag = defaultCountryTag()
                }
                applyPendingSharedIfNeeded()
            }
            .onDisappear {
                appState.isPostingSpot = false
            }
            .onChange(of: appState.pendingOfficialUrl) { _, newValue in
                guard let newValue = newValue else {
                    return
                }
                sharedUrlCandidate = newValue
                showSharedUrlAlert = true
            }
            .onChange(of: appState.pendingSharedLocation) { _, newValue in
                guard let newValue = newValue else {
                    return
                }
                sharedLocationCandidate = newValue
                sharedLocationName = appState.pendingSharedLocationName ?? ""
                showSharedLocationAlert = true
            }
            .onChange(of: appState.pendingSharedLocationError) { _, newValue in
                guard let newValue = newValue else {
                    return
                }
                sharedLocationErrorMessage = newValue
                showSharedLocationErrorAlert = true
            }
            .alert("公式サイトURLを追加しますか？", isPresented: $showSharedUrlAlert) {
                Button("追加") {
                    parkOfficialUrl = sharedUrlCandidate
                    appState.pendingOfficialUrl = nil
                }
                Button("キャンセル", role: .cancel) {
                    appState.pendingOfficialUrl = nil
                }
            } message: {
                Text(sharedUrlCandidate)
            }
            .alert("位置情報を追加しますか？", isPresented: $showSharedLocationAlert) {
                Button("追加") {
                    if let location = sharedLocationCandidate {
                        selectedLocation = location
                    }
                    if name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
                       !sharedLocationName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                        name = sharedLocationName
                    }
                    appState.pendingSharedLocation = nil
                    appState.pendingSharedLocationName = nil
                }
                Button("キャンセル", role: .cancel) {
                    appState.pendingSharedLocation = nil
                    appState.pendingSharedLocationName = nil
                }
            } message: {
                if !sharedLocationName.isEmpty {
                    Text(sharedLocationName)
                } else if let location = sharedLocationCandidate {
                    Text("座標: \(location.lat), \(location.lng)")
                } else {
                    Text("位置情報が見つかりませんでした。")
                }
            }
            .alert("位置情報を取り込めませんでした", isPresented: $showSharedLocationErrorAlert) {
                Button("OK") {
                    appState.pendingSharedLocationError = nil
                }
            } message: {
                Text(sharedLocationErrorMessage)
            }
            .sheet(isPresented: $showImagePicker) {
                SdzImagePicker(maxSelection: remainingImageSlots) { newImages in
                    let newItems = newImages.map { SdzSpotImageItem(localImage: $0) }
                    imageItems.append(contentsOf: newItems)
                }
            }
            .sheet(isPresented: $showUrlPickerSheet) {
                SdzUrlPickerSheetView(initialUrl: parkOfficialUrl) { selectedUrl in
                    parkOfficialUrl = selectedUrl
                }
            }
            .onChange(of: selectedLocation) { _, newValue in
                guard let newValue = newValue else {
                    countryTag = defaultCountryTag()
                    return
                }
                Task {
                    await resolveCountryTag(for: newValue)
                }
            }
        }
    }

    private var imageSectionHeader: some View {
        HStack {
            Text("画像")
            Spacer()
            Text("\(imageItems.count)/\(maxImagesPerSpot)")
                .font(SdzTypography.caption1)
                .foregroundColor(imageItems.count >= maxImagesPerSpot ? .sdzError : .sdzTextSecondary)
        }
    }

    private var remainingImageSlots: Int {
        max(0, maxImagesPerSpot - imageItems.count)
    }

    private func submit() {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedName.isEmpty {
            submitMessage = "スポット名を入力してください。"
            return
        }
        guard appState.idToken != nil else {
            submitMessage = "ログインが必要です。"
            return
        }
        guard selectedLocation != nil else {
            submitMessage = "投稿位置が未選択です。スポット画面でピンを置いてから投稿してください。"
            return
        }
        if imageItems.count > maxImagesPerSpot {
            submitMessage = "画像は最大\(maxImagesPerSpot)枚までです。"
            return
        }
        if spotCategory == .park, let hoursError = validateBusinessHours() {
            submitMessage = hoursError
            return
        }

        let tags = buildTags()
        let combinedDescription = buildDescription()
        let parkAttributes = spotCategory == .park ? buildParkAttributes() : nil
        let streetAttributes = buildStreetAttributes()

        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
        isSubmitting = true
        submitMessage = nil

        Task {
            do {
                let uploadedUrls = try await uploadImagesIfNeeded(apiClient: apiClient)
                let input = SdzCreateSpotInput(
                    name: trimmedName,
                    description: combinedDescription,
                    location: selectedLocation,
                    tags: tags?.isEmpty == true ? nil : tags,
                    images: uploadedUrls.isEmpty ? nil : uploadedUrls,
                    parkAttributes: parkAttributes,
                    streetAttributes: streetAttributes
                )
                _ = try await apiClient.createSpot(input)
                await MainActor.run {
                    self.isSubmitting = false
                    self.submitMessage = "投稿が完了しました。"
                }
            } catch {
                let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
                await MainActor.run {
                    self.isSubmitting = false
                    self.submitMessage = message
                }
            }
        }
    }

    private func buildTags() -> [String]? {
        var userTags = tagsString
            .split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }

        let categoryTag = spotCategory.defaultTag
        var requiredTags = [categoryTag]
        if let countryTag = countryTag?.trimmingCharacters(in: .whitespacesAndNewlines),
           !countryTag.isEmpty {
            requiredTags.append(countryTag)
        }

        userTags = uniqueTags(from: userTags)
        requiredTags = uniqueTags(from: requiredTags.filter { !userTags.contains($0) })

        let maxUserTags = max(0, 10 - requiredTags.count)
        if userTags.count > maxUserTags {
            userTags = Array(userTags.prefix(maxUserTags))
        }

        let tags = userTags + requiredTags
        return tags.isEmpty ? nil : tags
    }

    private func buildDescription() -> String? {
        let base = description.trimmingCharacters(in: .whitespacesAndNewlines)
        return base.isEmpty ? nil : base
    }

    private func applyDraftLocationIfNeeded() {
        guard let draft = appState.draftPostLocation else {
            return
        }
        selectedLocation = draft
        appState.draftPostLocation = nil
    }

    private func applyPendingSharedIfNeeded() {
        if !showSharedLocationAlert, let pendingLocation = appState.pendingSharedLocation {
            sharedLocationCandidate = pendingLocation
            sharedLocationName = appState.pendingSharedLocationName ?? ""
            showSharedLocationAlert = true
            return
        }
        if !showSharedUrlAlert, let pendingUrl = appState.pendingOfficialUrl {
            sharedUrlCandidate = pendingUrl
            showSharedUrlAlert = true
            return
        }
        if !showSharedLocationErrorAlert, let pendingError = appState.pendingSharedLocationError {
            sharedLocationErrorMessage = pendingError
            showSharedLocationErrorAlert = true
        }
    }

    private func closeComposer() {
        appState.isPostComposerPresented = false
        dismiss()
    }

    private func resolveCountryTag(for location: SdzSpotLocation) async {
        if lastResolvedCountryLocation == location, countryTag != nil {
            return
        }
        lastResolvedCountryLocation = location
        await MainActor.run {
            countryTag = defaultCountryTag()
        }
    }

    private func defaultCountryTag() -> String? {
        Locale.current.region?.identifier.uppercased()
    }

    private func uniqueTags(from tags: [String]) -> [String] {
        var seen: Set<String> = []
        var results: [String] = []
        for tag in tags {
            if seen.insert(tag).inserted {
                results.append(tag)
            }
        }
        return results
    }

    private func buildParkAttributes() -> SdzSpotParkAttributes? {
        let officialUrl = parkOfficialUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        let accessInfo = parkAccessInfo.trimmingCharacters(in: .whitespacesAndNewlines)
        let phoneNumber = parkPhoneNumber.trimmingCharacters(in: .whitespacesAndNewlines)
        let businessHours = buildBusinessHours()

        let cleanedUrl = officialUrl.isEmpty ? nil : officialUrl
        let cleanedAccess = accessInfo.isEmpty ? nil : accessInfo
        let cleanedPhone = phoneNumber.isEmpty ? nil : phoneNumber

        if cleanedUrl == nil && cleanedAccess == nil && cleanedPhone == nil && businessHours == nil {
            return nil
        }

        return SdzSpotParkAttributes(
            officialUrl: cleanedUrl,
            businessHours: businessHours,
            accessInfo: cleanedAccess,
            phoneNumber: cleanedPhone
        )
    }

    private func buildBusinessHours() -> SdzSpotBusinessHours? {
        let cleanedNote = parkScheduleNote.trimmingCharacters(in: .whitespacesAndNewlines)
        let noteValue = cleanedNote.isEmpty ? nil : cleanedNote

        switch parkScheduleType {
        case .irregular, .schoolOnly, .manual:
            return SdzSpotBusinessHours(
                scheduleType: parkScheduleType,
                is24Hours: false,
                sameAsWeekday: true,
                weekday: nil,
                weekend: nil,
                note: noteValue
            )
        case .weekdayOnly:
            let weekdayRange = parkIs24Hours
                ? nil
                : SdzSpotTimeRange(
                    startMinutes: minutes(from: parkWeekdayStart),
                    endMinutes: minutes(from: parkWeekdayEnd)
                )
            return SdzSpotBusinessHours(
                scheduleType: parkScheduleType,
                is24Hours: parkIs24Hours,
                sameAsWeekday: true,
                weekday: weekdayRange,
                weekend: nil,
                note: noteValue
            )
        case .weekendOnly:
            let weekendRange = parkIs24Hours
                ? nil
                : SdzSpotTimeRange(
                    startMinutes: minutes(from: parkWeekendStart),
                    endMinutes: minutes(from: parkWeekendEnd)
                )
            return SdzSpotBusinessHours(
                scheduleType: parkScheduleType,
                is24Hours: parkIs24Hours,
                sameAsWeekday: true,
                weekday: nil,
                weekend: weekendRange,
                note: noteValue
            )
        case .regular:
            if parkIs24Hours {
                return SdzSpotBusinessHours(
                    scheduleType: parkScheduleType,
                    is24Hours: true,
                    sameAsWeekday: true,
                    weekday: nil,
                    weekend: nil,
                    note: noteValue
                )
            }

            let weekdayRange = SdzSpotTimeRange(
                startMinutes: minutes(from: parkWeekdayStart),
                endMinutes: minutes(from: parkWeekdayEnd)
            )
            let weekendRange: SdzSpotTimeRange? = parkSameAsWeekday
                ? nil
                : SdzSpotTimeRange(
                    startMinutes: minutes(from: parkWeekendStart),
                    endMinutes: minutes(from: parkWeekendEnd)
                )

            return SdzSpotBusinessHours(
                scheduleType: parkScheduleType,
                is24Hours: false,
                sameAsWeekday: parkSameAsWeekday,
                weekday: weekdayRange,
                weekend: weekendRange,
                note: noteValue
            )
        }
    }

    private func buildStreetAttributes() -> SdzStreetAttributes? {
        let material = resolvedSurfaceMaterial()
        let roughness = stringLevel(from: streetRoughnessLevel)
        let crack = stringLevel(from: streetCrackLevel)
        let difficulty = stringLevel(from: streetDifficultyLevel)
        let cleanedNotes = streetNotes.trimmingCharacters(in: .whitespacesAndNewlines)

        let surfaceMaterial = material
        let surfaceCondition: SdzStreetSurfaceCondition? = (roughness.isEmpty && crack.isEmpty)
            ? nil
            : SdzStreetSurfaceCondition(
                roughness: roughness.isEmpty ? nil : roughness,
                crack: crack.isEmpty ? nil : crack
            )
        let sections = buildStreetSections()
        let cleanedDifficulty = difficulty.isEmpty ? nil : difficulty
        let notes = cleanedNotes.isEmpty ? nil : cleanedNotes

        if surfaceMaterial == nil && surfaceCondition == nil && sections == nil && cleanedDifficulty == nil && notes == nil {
            return nil
        }

        return SdzStreetAttributes(
            surfaceMaterial: surfaceMaterial,
            surfaceCondition: surfaceCondition,
            sections: sections,
            difficulty: cleanedDifficulty,
            notes: notes
        )
    }

    private func buildStreetSections() -> [SdzStreetSection]? {
        let sections: [SdzStreetSection] = streetSections.compactMap { input in
            let type = input.type.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !type.isEmpty else {
                return nil
            }
            return SdzStreetSection(
                type: type,
                count: parseInt(input.count),
                heightCm: parseInt(input.heightCm),
                widthCm: parseInt(input.widthCm),
                notes: input.notes.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? nil
                    : input.notes.trimmingCharacters(in: .whitespacesAndNewlines)
            )
        }
        return sections.isEmpty ? nil : sections
    }

    private func validateBusinessHours() -> String? {
        switch parkScheduleType {
        case .irregular, .schoolOnly, .manual:
            let cleanedNote = parkScheduleNote.trimmingCharacters(in: .whitespacesAndNewlines)
            if cleanedNote.isEmpty {
                return "営業時間の補足を入力してください。"
            }
            return nil
        case .weekdayOnly:
            guard !parkIs24Hours else {
                return nil
            }
            let weekdayStart = minutes(from: parkWeekdayStart)
            let weekdayEnd = minutes(from: parkWeekdayEnd)
            if weekdayStart >= weekdayEnd {
                return "平日の営業時間が正しくありません。"
            }
            return nil
        case .weekendOnly:
            guard !parkIs24Hours else {
                return nil
            }
            let weekendStart = minutes(from: parkWeekendStart)
            let weekendEnd = minutes(from: parkWeekendEnd)
            if weekendStart >= weekendEnd {
                return "週末の営業時間が正しくありません。"
            }
            return nil
        case .regular:
            guard !parkIs24Hours else {
                return nil
            }
            let weekdayStart = minutes(from: parkWeekdayStart)
            let weekdayEnd = minutes(from: parkWeekdayEnd)
            if weekdayStart >= weekdayEnd {
                return "平日の営業時間が正しくありません。"
            }
            if !parkSameAsWeekday {
                let weekendStart = minutes(from: parkWeekendStart)
                let weekendEnd = minutes(from: parkWeekendEnd)
                if weekendStart >= weekendEnd {
                    return "週末の営業時間が正しくありません。"
                }
            }
            return nil
        }
    }

    private func minutes(from date: Date) -> Int {
        let components = Calendar.current.dateComponents([.hour, .minute], from: date)
        return (components.hour ?? 0) * 60 + (components.minute ?? 0)
    }

    private func resolvedSurfaceMaterial() -> String? {
        switch streetSurfaceMaterialOption {
        case "未入力":
            return nil
        case "その他":
            let cleaned = streetSurfaceMaterialCustom.trimmingCharacters(in: .whitespacesAndNewlines)
            return cleaned.isEmpty ? "その他" : cleaned
        default:
            return streetSurfaceMaterialOption
        }
    }

    private func stringLevel(from level: Int?) -> String {
        guard let level else {
            return ""
        }
        return String(level)
    }

    @ViewBuilder
    private func ratingPicker(title: String, selection: Binding<Int?>, helpText: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(title)
                Spacer()
                Picker(title, selection: selection) {
                    Text("未入力").tag(Optional<Int>.none)
                    ForEach(1...5, id: \.self) { value in
                        Text("\(value)").tag(Int?.some(value))
                    }
                }
                .labelsHidden()
                .pickerStyle(.menu)
            }
            Text(helpText)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }

    private func parseInt(_ value: String) -> Int? {
        Int(value.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    private func uploadImagesIfNeeded(apiClient: SdzApiClient) async throws -> [String] {
        let imagesToUpload = imageItems.compactMap { $0.localImage }
        guard !imagesToUpload.isEmpty else {
            return []
        }

        var uploadedUrls: [String] = []
        for image in imagesToUpload {
            guard let payload = SdzImageOptimizer.optimize(image) else {
                throw SdzPostError.imageEncodingFailed
            }
            let upload = try await apiClient.requestUploadUrl(contentType: payload.contentType)
            try await apiClient.uploadImage(
                data: payload.data,
                contentType: payload.contentType,
                uploadUrl: upload.uploadUrl
            )
            uploadedUrls.append(upload.objectUrl)
        }
        return uploadedUrls
    }
}

private enum SdzPostError: LocalizedError {
    case imageEncodingFailed

    var errorDescription: String? {
        switch self {
        case .imageEncodingFailed:
            return "画像の変換に失敗しました。"
        }
    }
}

enum SdzSpotImageSource {
    case local(UIImage)
    case remote(url: String)
}

struct SdzSpotImageItem: Identifiable {
    let id: UUID
    let source: SdzSpotImageSource

    init(localImage: UIImage) {
        id = UUID()
        source = .local(localImage)
    }

    init(remoteUrl: String) {
        id = UUID()
        source = .remote(url: remoteUrl)
    }

    var localImage: UIImage? {
        if case let .local(image) = source {
            return image
        }
        return nil
    }

    var remoteUrl: URL? {
        if case let .remote(url) = source {
            return URL(string: url)
        }
        return nil
    }
}

struct SdzSpotImageRow: View {
    @Binding var items: [SdzSpotImageItem]
    let maxImages: Int
    let onAdd: () -> Void
    @State private var draggedItemId: UUID?
    @State private var dragOffset: CGFloat = 0
    private let itemSize: CGFloat = 90
    private let itemSpacing: CGFloat = 12

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: itemSpacing) {
                ForEach(items) { item in
                    let isDragging = draggedItemId == item.id
                    let isMain = items.first?.id == item.id
                    SdzSpotImageThumbnail(
                        item: item,
                        isMain: isMain,
                        isDragging: isDragging,
                        onRemove: { remove(item) }
                    )
                    .frame(width: itemSize, height: itemSize)
                    .offset(x: isDragging ? dragOffset : 0)
                    .gesture(reorderGesture(for: item))
                }

                if items.count < maxImages {
                    Button(action: onAdd) {
                        VStack(spacing: 6) {
                            Image(systemName: "plus")
                                .font(.title2)
                            Text("追加")
                                .font(.caption)
                        }
                        .frame(width: itemSize, height: itemSize)
                        .foregroundColor(.accentColor)
                        .background(Color(.secondarySystemBackground))
                        .cornerRadius(10)
                    }
                }
            }
            .padding(.vertical, 4)
        }
    }

    private func remove(_ item: SdzSpotImageItem) {
        items.removeAll { $0.id == item.id }
    }

    private func reorderGesture(for item: SdzSpotImageItem) -> some Gesture {
        let longPress = LongPressGesture(minimumDuration: 0.18)
        let drag = DragGesture()
        let stride = itemSize + itemSpacing

        return longPress
            .sequenced(before: drag)
            .onChanged { value in
                switch value {
                case .first(true):
                    draggedItemId = item.id
                case .second(true, let dragValue?):
                    draggedItemId = item.id
                    dragOffset = dragValue.translation.width
                default:
                    break
                }
            }
            .onEnded { value in
                defer {
                    dragOffset = 0
                    draggedItemId = nil
                }
                guard case .second(true, let dragValue?) = value,
                      let fromIndex = items.firstIndex(where: { $0.id == item.id })
                else {
                    return
                }

                let shift = Int((dragValue.translation.width / stride).rounded())
                if shift == 0 {
                    return
                }

                let toIndex = max(0, min(items.count - 1, fromIndex + shift))
                guard toIndex != fromIndex else {
                    return
                }
                withAnimation {
                    items.move(
                        fromOffsets: IndexSet(integer: fromIndex),
                        toOffset: toIndex > fromIndex ? toIndex + 1 : toIndex
                    )
                }
            }
    }
}

private struct SdzSpotImageThumbnail: View {
    let item: SdzSpotImageItem
    let isMain: Bool
    let isDragging: Bool
    let onRemove: () -> Void

    var body: some View {
        ZStack(alignment: .topTrailing) {
            ZStack(alignment: .topLeading) {
                imageView
                if isMain {
                    Text("メイン")
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .foregroundColor(.white)
                        .background(Color.black.opacity(0.6))
                        .clipShape(Capsule())
                        .padding(6)
                }
            }
            Button(action: onRemove) {
                Image(systemName: "xmark.circle.fill")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundColor(.white)
                    .padding(4)
                    .background(Circle().fill(Color.black.opacity(0.7)))
                    .shadow(color: Color.black.opacity(0.2), radius: 2, x: 0, y: 1)
            }
            .padding(4)
        }
        .frame(width: 90, height: 90)
        .background(Color(.secondarySystemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .shadow(color: Color.black.opacity(isDragging ? 0.25 : 0.0), radius: 6, x: 0, y: 3)
        .scaleEffect(isDragging ? 1.05 : 1.0)
        .rotationEffect(isDragging ? Angle(degrees: -1.2) : .zero)
        .animation(.easeInOut(duration: 0.15), value: isDragging)
        .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
    }

    @ViewBuilder
    private var imageView: some View {
        if let image = item.localImage {
            Image(uiImage: image)
                .resizable()
                .scaledToFill()
                .frame(width: 90, height: 90)
                .clipped()
        } else if let url = item.remoteUrl {
            AsyncImage(url: url) { phase in
                switch phase {
                case .empty:
                    ProgressView()
                        .frame(width: 90, height: 90)
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
            .frame(width: 90, height: 90)
            .clipped()
        } else {
            Color.gray.opacity(0.2)
                .frame(width: 90, height: 90)
        }
    }
}


// A multi-image picker for selecting photos from the photo library.
struct SdzImagePicker: UIViewControllerRepresentable {
    @Environment(\.dismiss) private var dismiss
    let maxSelection: Int
    let onPick: ([UIImage]) -> Void

    func makeUIViewController(context: Context) -> PHPickerViewController {
        var configuration = PHPickerConfiguration(photoLibrary: .shared())
        configuration.filter = .images
        configuration.selectionLimit = max(1, maxSelection)

        let picker = PHPickerViewController(configuration: configuration)
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: PHPickerViewController, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    final class Coordinator: NSObject, PHPickerViewControllerDelegate {
        private let parent: SdzImagePicker

        init(_ parent: SdzImagePicker) {
            self.parent = parent
        }

        func picker(_ picker: PHPickerViewController, didFinishPicking results: [PHPickerResult]) {
            guard !results.isEmpty else {
                self.parent.dismiss()
                return
            }

            var loadedImages = Array<UIImage?>(repeating: nil, count: results.count)
            let group = DispatchGroup()

            for (index, result) in results.enumerated() {
                guard result.itemProvider.canLoadObject(ofClass: UIImage.self) else {
                    continue
                }
                group.enter()
                result.itemProvider.loadObject(ofClass: UIImage.self) { object, _ in
                    if let image = object as? UIImage {
                        loadedImages[index] = image
                    }
                    group.leave()
                }
            }

            group.notify(queue: .main) {
                let images = loadedImages.compactMap { $0 }
                let limitedImages = self.parent.maxSelection > 0
                    ? Array(images.prefix(self.parent.maxSelection))
                    : []
                self.parent.onPick(limitedImages)
                self.parent.dismiss()
            }
        }
    }
}

#if DEBUG
struct PostView_Previews: PreviewProvider {
    static var previews: some View {
        PostView()
            .environmentObject(SdzAppState())
    }
}
#endif
