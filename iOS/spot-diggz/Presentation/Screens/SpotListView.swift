import SwiftUI
import MapKit

private enum SdzSpotListCategory: String, CaseIterable, Identifiable {
    case all
    case park
    case street

    var id: String { rawValue }

    var label: String {
        switch self {
        case .all:
            return "すべて"
        case .park:
            return "パーク"
        case .street:
            return "ストリート"
        }
    }
}

private struct SdzSpotAddressSummary: Sendable {
    let countryName: String?
    let countryCode: String?
    let adminArea: String?
    let compactAddress: String
    let searchableAddressText: String

    init?(mapItem: MKMapItem) {
        guard let address = mapItem.address else { return nil }
        let placemark = mapItem.placemark
        self.countryName = placemark.country
        self.countryCode = placemark.isoCountryCode
        self.adminArea = placemark.administrativeArea
        self.compactAddress = address.shortAddress ?? address.fullAddress ?? ""
        self.searchableAddressText = (address.fullAddress ?? "").lowercased()
    }
}

/// Displays all registered spots in a list format with location filters.
struct SpotListView: View {
    @EnvironmentObject var appState: SdzAppState

    @State private var spots: [SdzSpot] = []
    @State private var addressBySpotId: [String: SdzSpotAddressSummary] = [:]
    @State private var geocodeTask: Task<Void, Never>?
    @State private var isLoading: Bool = false
    @State private var errorMessage: String?
    @State private var selectedCategory: SdzSpotListCategory = .all
    @State private var countryFilter: String = ""
    @State private var prefectureFilter: String = ""
    @State private var addressFilter: String = ""

    var body: some View {
        NavigationView {
            List {
                filterSection
                contentSection
            }
            .navigationTitle("一覧")
            .refreshable {
                await reloadSpots()
            }
            .task {
                if spots.isEmpty {
                    await reloadSpots()
                }
            }
            .onDisappear {
                geocodeTask?.cancel()
            }
        }
    }

    private var filterSection: some View {
        Section(header: Text("絞り込み")) {
            Picker("種別", selection: $selectedCategory) {
                ForEach(SdzSpotListCategory.allCases) { category in
                    Text(category.label).tag(category)
                }
            }
            .pickerStyle(.segmented)

            TextField("国 (例: Japan / JP)", text: $countryFilter)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled(true)
            TextField("都道府県 (例: Osaka)", text: $prefectureFilter)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled(true)
            TextField("住所 (例: 堺市中区 / バラ池公園)", text: $addressFilter)
                .textInputAutocapitalization(.never)

            Button("フィルタをクリア") {
                countryFilter = ""
                prefectureFilter = ""
                addressFilter = ""
                selectedCategory = .all
            }
            .foregroundColor(.accentColor)
        }
    }

    @ViewBuilder
    private var contentSection: some View {
        Section(header: Text("登録スポット")) {
            if isLoading {
                ProgressView("読み込み中...")
            } else if let errorMessage {
                VStack(alignment: .leading, spacing: 8) {
                    Text(errorMessage)
                        .foregroundColor(.red)
                    Button("再読み込み") {
                        Task {
                            await reloadSpots()
                        }
                    }
                    .buttonStyle(.bordered)
                }
            } else if filteredSpots.isEmpty {
                Text("条件に一致するスポットがありません。")
                    .foregroundColor(.secondary)
            } else {
                ForEach(filteredSpots) { spot in
                    NavigationLink(destination: SpotDetailView(spot: spot)) {
                        VStack(alignment: .leading, spacing: 6) {
                            SpotCardView(spot: spot)
                            if let address = addressBySpotId[spot.spotId]?.compactAddress, !address.isEmpty {
                                Text(address)
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                            } else if let location = spot.location {
                                Text("lat: \(location.lat), lng: \(location.lng)")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                            }
                        }
                    }
                }
            }
        }
    }

    private var filteredSpots: [SdzSpot] {
        let countryQuery = normalized(countryFilter)
        let prefectureQuery = normalized(prefectureFilter)
        let addressQuery = normalized(addressFilter)

        return spots.filter { spot in
            guard matchesCategory(spot) else {
                return false
            }
            let summary = addressBySpotId[spot.spotId]
            let searchableAddress = summary?.searchableAddressText ?? ""
            let searchableSpotText = [
                spot.name,
                spot.description ?? "",
                spot.tags.joined(separator: " ")
            ]
            .joined(separator: " ")
            .lowercased()

            let searchableCountry = [
                summary?.countryName ?? "",
                summary?.countryCode ?? "",
                spot.tags.joined(separator: " ")
            ]
            .joined(separator: " ")
            .lowercased()
            let searchablePrefecture = (summary?.adminArea ?? "").lowercased()

            if !countryQuery.isEmpty && !searchableCountry.contains(countryQuery) {
                return false
            }
            if !prefectureQuery.isEmpty && !searchablePrefecture.contains(prefectureQuery) {
                return false
            }
            if !addressQuery.isEmpty &&
                !searchableAddress.contains(addressQuery) &&
                !searchableSpotText.contains(addressQuery) {
                return false
            }
            return true
        }
    }

    private func matchesCategory(_ spot: SdzSpot) -> Bool {
        switch selectedCategory {
        case .all:
            return true
        case .park:
            return isParkSpot(spot)
        case .street:
            return !isParkSpot(spot)
        }
    }

    private func isParkSpot(_ spot: SdzSpot) -> Bool {
        spot.sdzIsPark
    }

    private func normalized(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func reloadSpots() async {
        await MainActor.run {
            isLoading = true
            errorMessage = nil
        }

        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
        do {
            let fetched = try await apiClient.fetchSpots(includeAuth: true)
            let sorted = fetched.sorted { $0.updatedAt > $1.updatedAt }
            await MainActor.run {
                spots = sorted
                isLoading = false
                let ids = Set(sorted.map(\.spotId))
                addressBySpotId = addressBySpotId.filter { ids.contains($0.key) }
                startReverseGeocoding(for: sorted)
            }
        } catch {
            let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
            await MainActor.run {
                errorMessage = message
                isLoading = false
            }
        }
    }

    private func startReverseGeocoding(for spots: [SdzSpot]) {
        geocodeTask?.cancel()
        let unresolved = spots.compactMap { spot -> (String, SdzSpotLocation)? in
            guard addressBySpotId[spot.spotId] == nil,
                  let location = spot.location else {
                return nil
            }
            return (spot.spotId, location)
        }
        guard !unresolved.isEmpty else {
            return
        }

        geocodeTask = Task {
            for (spotId, location) in unresolved {
                guard !Task.isCancelled else {
                    return
                }
                if let summary = await reverseGeocode(location: location) {
                    await MainActor.run {
                        addressBySpotId[spotId] = summary
                    }
                }
                try? await Task.sleep(nanoseconds: 120_000_000)
            }
        }
    }

    private func reverseGeocode(location: SdzSpotLocation) async -> SdzSpotAddressSummary? {
        let coordinate = CLLocation(latitude: location.lat, longitude: location.lng)
        guard let request = MKReverseGeocodingRequest(location: coordinate) else {
            return nil
        }
        do {
            let mapItems = try await request.mapItems
            guard let first = mapItems.first else { return nil }
            return SdzSpotAddressSummary(mapItem: first)
        } catch {
            return nil
        }
    }
}

#if DEBUG
struct SpotListView_Previews: PreviewProvider {
    static var previews: some View {
        SpotListView()
            .environmentObject(SdzAppState())
    }
}
#endif
