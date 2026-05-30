import SwiftUI
import MapKit
import Combine

private struct SearchBarHeightKey: PreferenceKey {
    static var defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = max(value, nextValue())
    }
}

private struct MapSizeKey: PreferenceKey {
    static var defaultValue: CGSize = .zero

    static func reduce(value: inout CGSize, nextValue: () -> CGSize) {
        let next = nextValue()
        if next != .zero {
            value = next
        }
    }
}

/// Map-based location picker for selecting a spot coordinate.
struct SdzLocationPickerView: View {
    enum Mode: Equatable {
        case embedded
        case fullScreen
    }

    @EnvironmentObject var appState: SdzAppState
    @Binding var selectedLocation: SdzSpotLocation?
    var height: CGFloat = 320
    var mode: Mode = .embedded
    var onRequestCurrentLocation: (() -> Void)? = nil
    var onRequestExpandedSelection: (() -> Void)? = nil
    var onConfirmSelection: ((SdzSpotLocation) -> Void)? = nil
    var expandedSelectionIconName: String = "plus.magnifyingglass"

    @State private var cameraPosition: MapCameraPosition = .automatic
    @StateObject private var searchViewModel = SdzLocationSearchViewModel()
    @State private var searchBarHeight: CGFloat = 0
    @State private var hideSearchResults: Bool = false
    @State private var mapSize: CGSize = .zero
    @State private var hasSearchFailure: Bool = false
    @State private var visibleRegion: MKCoordinateRegion = Self.defaultRegion
    @State private var workingLocation: SdzSpotLocation?
    @State private var allSpots: [SdzSpot] = []
    @State private var nearbySpots: [SdzSpot] = []
    @State private var selectedNearbySpot: SdzSpot?
    @State private var isFetchingNearbySpots: Bool = false
    @State private var nearbyFetchTask: Task<Void, Never>?

    var body: some View {
        ZStack(alignment: .top) {
            VStack(spacing: 8) {
                if #available(iOS 17.0, *) {
                    MapReader { proxy in
                        Map(position: $cameraPosition) {
                            if !isFullScreenMode, let location = currentSelection {
                                Marker(
                                    "選択位置",
                                    coordinate: CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng)
                                )
                            }
                            if isFullScreenMode {
                                ForEach(nearbySpots) { spot in
                                    if let coordinate = coordinate(for: spot.location) {
                                        Annotation("", coordinate: coordinate) {
                                            Button {
                                                selectedNearbySpot = spot
                                            } label: {
                                                Image(systemName: "mappin.circle.fill")
                                                    .font(.title2)
                                                    .foregroundColor(nearbySpotTintColor(for: spot))
                                                    .shadow(color: Color.black.opacity(0.2), radius: 2, x: 0, y: 1)
                                            }
                                            .buttonStyle(.plain)
                                            .accessibilityLabel("\(spot.name)のスポット情報")
                                        }
                                    }
                                }
                            }
                        }
                        .onMapCameraChange { context in
                            visibleRegion = context.region
                            searchViewModel.updateRegion(context.region, source: .map)
                            if isFullScreenMode {
                                workingLocation = SdzSpotLocation(
                                    lat: context.region.center.latitude,
                                    lng: context.region.center.longitude
                                )
                                updateNearbySpots(for: context.region)
                            }
                        }
                        .overlay(alignment: .bottomTrailing) {
                            VStack(spacing: 10) {
                                if let onRequestCurrentLocation {
                                    Button {
                                        onRequestCurrentLocation()
                                    } label: {
                                        Image(systemName: "location.fill")
                                            .font(.system(size: 16, weight: .semibold))
                                            .frame(width: 36, height: 36)
                                            .background(.ultraThinMaterial)
                                            .clipShape(Circle())
                                    }
                                    .accessibilityLabel("現在地に移動")
                                }
                                if let onRequestExpandedSelection {
                                    Button {
                                        onRequestExpandedSelection()
                                    } label: {
                                        Image(systemName: expandedSelectionIconName)
                                            .font(.system(size: 16, weight: .semibold))
                                            .frame(width: 36, height: 36)
                                            .background(.ultraThinMaterial)
                                            .clipShape(Circle())
                                    }
                                    .accessibilityLabel("拡大して選択")
                                }
                            }
                            .padding(.trailing, 12)
                            .padding(.bottom, 12)
                        }
                        .overlay(alignment: .topLeading) {
                            if isFullScreenMode {
                                Text("地図を動かして中央の十字で位置を合わせてください")
                                    .font(.caption)
                                    .padding(8)
                                    .background(.ultraThinMaterial)
                                    .cornerRadius(8)
                                    .padding(8)
                            } else if currentSelection == nil {
                                Text("地図をダブルタップして位置を選択")
                                    .font(.caption)
                                    .padding(8)
                                    .background(.ultraThinMaterial)
                                    .cornerRadius(8)
                                    .padding(8)
                            }
                        }
                        .overlay(alignment: .center) {
                            if isFullScreenMode {
                                Image(systemName: "plus")
                                    .font(.system(size: 22, weight: .bold))
                                    .foregroundColor(.primary)
                                    .padding(8)
                                    .background(.ultraThinMaterial, in: Circle())
                                    .shadow(color: Color.black.opacity(0.12), radius: 3, x: 0, y: 1)
                                    .accessibilityHidden(true)
                            }
                        }
                        .overlay(alignment: .bottom) {
                            if isFullScreenMode {
                                VStack(spacing: 10) {
                                    if isFetchingNearbySpots {
                                        HStack(spacing: 8) {
                                            ProgressView()
                                                .scaleEffect(0.9)
                                            Text("近くの既存スポットを読み込み中")
                                                .font(.caption)
                                                .foregroundColor(.secondary)
                                        }
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 8)
                                        .background(.ultraThinMaterial, in: Capsule())
                                    }
                                    if let spot = selectedNearbySpot {
                                        nearbySpotCard(for: spot)
                                    }
                                    Button {
                                        commitFullScreenSelection()
                                    } label: {
                                        Text("この位置で決定")
                                            .font(.headline)
                                            .frame(maxWidth: .infinity)
                                            .padding(.vertical, 14)
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .disabled(workingLocation == nil)
                                }
                                .padding(.horizontal, 12)
                                .padding(.bottom, 12)
                            }
                        }
                        .simultaneousGesture(
                            SpatialTapGesture(count: 2)
                                .onEnded { value in
                                    if let coordinate = proxy.convert(value.location, from: .local) {
                                        updateSelection(with: coordinate)
                                    }
                                }
                        )
                    }
                } else {
                    Map(
                        coordinateRegion: .constant(defaultRegion),
                        annotationItems: currentSelection.map { [$0] } ?? []
                    ) { location in
                        MapMarker(
                            coordinate: CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng)
                        )
                    }
                    .overlay(alignment: .bottomTrailing) {
                        VStack(spacing: 10) {
                            if let onRequestCurrentLocation {
                                Button {
                                    onRequestCurrentLocation()
                                } label: {
                                    Image(systemName: "location.fill")
                                        .font(.system(size: 16, weight: .semibold))
                                        .frame(width: 36, height: 36)
                                        .background(.ultraThinMaterial)
                                        .clipShape(Circle())
                                }
                                .accessibilityLabel("現在地に移動")
                            }
                            if let onRequestExpandedSelection {
                                Button {
                                    onRequestExpandedSelection()
                                } label: {
                                    Image(systemName: expandedSelectionIconName)
                                        .font(.system(size: 16, weight: .semibold))
                                        .frame(width: 36, height: 36)
                                        .background(.ultraThinMaterial)
                                        .clipShape(Circle())
                                }
                                .accessibilityLabel("拡大して選択")
                            }
                        }
                        .padding(.trailing, 12)
                        .padding(.bottom, 12)
                    }
                }
            }
            searchOverlay
        }
        .frame(maxWidth: .infinity)
        .frame(height: isFullScreenMode ? nil : height)
        .frame(maxHeight: isFullScreenMode ? .infinity : nil)
        .background(
            GeometryReader { proxy in
                Color.clear
                    .preference(key: MapSizeKey.self, value: proxy.size)
            }
        )
        .onPreferenceChange(SearchBarHeightKey.self) { height in
            searchBarHeight = height
        }
        .onPreferenceChange(MapSizeKey.self) { size in
            mapSize = size
        }
        .onAppear {
            workingLocation = selectedLocation
            updateCameraPosition()
            loadNearbySpotsIfNeeded()
        }
        .onDisappear {
            nearbyFetchTask?.cancel()
        }
        .onChangeCompat(of: selectedLocation) {
            if !isFullScreenMode {
                updateCameraPosition()
            }
        }
        .onChangeCompat(of: searchViewModel.query) {
            hideSearchResults = false
            hasSearchFailure = false
        }
    }

    private var isFullScreenMode: Bool {
        mode == .fullScreen
    }

    private var currentSelection: SdzSpotLocation? {
        isFullScreenMode ? workingLocation : selectedLocation
    }

    private var searchBarMaxWidth: CGFloat {
        let width = mapSize.width
        guard width > 0 else { return 300 }
        let reservedRight: CGFloat = 68
        let maxWidth = width - reservedRight - 24
        return max(220, min(maxWidth, width - 24))
    }

    private static let defaultRegion = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 35.6812, longitude: 139.7671),
        span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
    )

    private var defaultRegion: MKCoordinateRegion { Self.defaultRegion }

    private func updateCameraPosition() {
        if let location = currentSelection {
            let coordinate = CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng)
            let region = MKCoordinateRegion(
                center: coordinate,
                span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
            )
            cameraPosition = .region(region)
            visibleRegion = region
            searchViewModel.updateRegion(region, source: .selection)
            if isFullScreenMode {
                updateNearbySpots(for: region)
            }
        } else {
            cameraPosition = .region(defaultRegion)
            visibleRegion = defaultRegion
            searchViewModel.resetRegionToDefault()
            if isFullScreenMode {
                workingLocation = SdzSpotLocation(
                    lat: defaultRegion.center.latitude,
                    lng: defaultRegion.center.longitude
                )
                updateNearbySpots(for: defaultRegion)
            }
        }
    }

    private func updateSelection(with coordinate: CLLocationCoordinate2D) {
        let location = SdzSpotLocation(lat: coordinate.latitude, lng: coordinate.longitude)
        if isFullScreenMode {
            workingLocation = location
            selectedNearbySpot = nil
        } else {
            selectedLocation = location
        }
        let region = MKCoordinateRegion(
            center: coordinate,
            span: MKCoordinateSpan(latitudeDelta: 0.01, longitudeDelta: 0.01)
        )
        cameraPosition = .region(region)
        visibleRegion = region
        searchViewModel.updateRegion(region, source: .selection)
        if isFullScreenMode {
            updateNearbySpots(for: region)
        }
    }

    private var searchOverlay: some View {
        VStack(spacing: 6) {
            GeometryReader { proxy in
                let availableWidth = max(0, proxy.size.width - 16)
                let maxWidth = min(availableWidth, searchBarMaxWidth)
                HStack(spacing: 8) {
                    TextField("場所を検索", text: $searchViewModel.query)
                        .textInputAutocapitalization(.never)
                        .disableAutocorrection(true)
                        .submitLabel(.search)
                        .onSubmit {
                            searchViewModel.searchDirect { coordinate, resolvedName in
                                guard let coordinate = coordinate else {
                                    hasSearchFailure = true
                                    return
                                }
                                updateSelection(with: coordinate)
                                searchViewModel.applyDirectSearchResult(name: resolvedName)
                                hasSearchFailure = false
                                hideSearchResults = true
                            }
                        }
                    if !searchViewModel.query.isEmpty {
                        Button {
                            searchViewModel.reset()
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .frame(width: max(1, maxWidth), alignment: .leading)
                .background(.ultraThinMaterial)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .shadow(color: Color.black.opacity(0.08), radius: 4, x: 0, y: 2)
                .background(
                    GeometryReader { innerProxy in
                        Color.clear
                            .preference(key: SearchBarHeightKey.self, value: innerProxy.size.height)
                    }
                )
            }
            .frame(height: 44)
            .frame(maxWidth: .infinity, alignment: .leading)

            if searchViewModel.shouldShowResults && !hideSearchResults {
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        if searchViewModel.shouldShowDirectSearch {
                            Button {
                                searchViewModel.searchDirect { coordinate, resolvedName in
                                    guard let coordinate = coordinate else {
                                        hasSearchFailure = true
                                        return
                                    }
                                    updateSelection(with: coordinate)
                                    searchViewModel.applyDirectSearchResult(name: resolvedName)
                                    hasSearchFailure = false
                                    hideSearchResults = true
                                }
                            } label: {
                                HStack(spacing: 8) {
                                    Image(systemName: "magnifyingglass")
                                        .foregroundColor(.secondary)
                                    Text("「\(searchViewModel.query)」で検索")
                                        .font(.subheadline)
                                        .foregroundColor(.primary)
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8)
                                .padding(.horizontal, 12)
                            }
                            if !searchViewModel.results.isEmpty {
                                Divider()
                            }
                        }
                        if !searchViewModel.searchItems.isEmpty {
                            ForEach(Array(searchViewModel.searchItems.enumerated()), id: \.offset) { index, item in
                                Button {
                                    updateSelection(with: item.location.coordinate)
                                    searchViewModel.applyDirectSearchResult(name: item.name)
                                    hideSearchResults = true
                                } label: {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(item.name ?? "検索結果")
                                            .font(.subheadline)
                                            .foregroundColor(.primary)
                                    }
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(.vertical, 8)
                                    .padding(.horizontal, 12)
                                }
                                if index < searchViewModel.searchItems.count - 1 {
                                    Divider()
                                }
                            }
                            if !searchViewModel.results.isEmpty {
                                Divider()
                            }
                        }
                        ForEach(searchViewModel.results, id: \.self) { completion in
                            Button {
                                searchViewModel.resolve(completion: completion) { coordinate in
                                    guard let coordinate = coordinate else {
                                        return
                                    }
                                    updateSelection(with: coordinate)
                                    searchViewModel.applySelection(completion)
                                    hideSearchResults = true
                                }
                            } label: {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(completion.title)
                                        .font(.subheadline)
                                        .foregroundColor(.primary)
                                    if !completion.subtitle.isEmpty {
                                        Text(completion.subtitle)
                                            .font(.caption)
                                            .foregroundColor(.secondary)
                                    }
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8)
                                .padding(.horizontal, 12)
                            }
                            if completion != searchViewModel.results.last {
                                Divider()
                            }
                        }
                        if hasSearchFailure
                            && searchViewModel.searchItems.isEmpty
                            && searchViewModel.results.isEmpty {
                            Divider()
                            HStack(spacing: 8) {
                                Image(systemName: "exclamationmark.triangle")
                                    .foregroundColor(.secondary)
                                Text("候補なし。地図を動かして中央で位置を決定してください。")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.vertical, 10)
                            .padding(.horizontal, 12)
                        }
                    }
                }
                .frame(maxHeight: 180)
                .background(.ultraThinMaterial)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .shadow(color: Color.black.opacity(0.08), radius: 4, x: 0, y: 2)
            }
        }
        .padding(8)
    }

    @ViewBuilder
    private func nearbySpotCard(for spot: SdzSpot) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("既存スポット")
                .font(.caption)
                .foregroundColor(.secondary)
            Text(spot.name)
                .font(.subheadline.weight(.semibold))
                .foregroundColor(.primary)
                .lineLimit(2)
            HStack(spacing: 8) {
                Text(isParkSpot(spot) ? "スケートパーク" : "ストリート")
                    .font(.caption2)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(nearbySpotTintColor(for: spot).opacity(0.15), in: Capsule())
                if let distanceText = distanceText(from: visibleRegion.center, to: spot.location) {
                    Text(distanceText)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private func commitFullScreenSelection() {
        guard isFullScreenMode, let workingLocation else {
            return
        }
        selectedLocation = workingLocation
        onConfirmSelection?(workingLocation)
    }

    private func loadNearbySpotsIfNeeded() {
        guard isFullScreenMode else {
            return
        }
        nearbyFetchTask?.cancel()
        guard !isFetchingNearbySpots else {
            return
        }
        isFetchingNearbySpots = true
        nearbyFetchTask = Task {
            let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
            do {
                let fetched = try await apiClient.fetchSpots(includeAuth: true)
                guard !Task.isCancelled else {
                    return
                }
                await MainActor.run {
                    allSpots = fetched.filter { $0.location != nil }
                    isFetchingNearbySpots = false
                    updateNearbySpots(for: visibleRegion)
                }
            } catch {
                guard !Task.isCancelled else {
                    return
                }
                await MainActor.run {
                    allSpots = []
                    nearbySpots = []
                    selectedNearbySpot = nil
                    isFetchingNearbySpots = false
                }
            }
        }
    }

    private func updateNearbySpots(for region: MKCoordinateRegion) {
        guard isFullScreenMode else {
            return
        }
        let latOffset = max(region.span.latitudeDelta * 0.7, 0.01)
        let lngOffset = max(region.span.longitudeDelta * 0.7, 0.01)
        let minLat = region.center.latitude - latOffset
        let maxLat = region.center.latitude + latOffset
        let minLng = region.center.longitude - lngOffset
        let maxLng = region.center.longitude + lngOffset
        let center = CLLocation(latitude: region.center.latitude, longitude: region.center.longitude)
        let nearbyPairs: [(spot: SdzSpot, distance: CLLocationDistance)] = allSpots
            .compactMap { spot in
                guard let location = spot.location else { return nil }
                guard (minLat...maxLat).contains(location.lat),
                      (minLng...maxLng).contains(location.lng) else {
                    return nil
                }
                let distance = center.distance(
                    from: CLLocation(latitude: location.lat, longitude: location.lng)
                )
                return (spot: spot, distance: distance)
            }
            .sorted { $0.distance < $1.distance }
        nearbySpots = nearbyPairs.prefix(40).map { $0.spot }
        if let selectedNearbySpot,
           !nearbySpots.contains(where: { $0.spotId == selectedNearbySpot.spotId }) {
            self.selectedNearbySpot = nil
        }
    }

    private func distanceText(from center: CLLocationCoordinate2D, to location: SdzSpotLocation?) -> String? {
        guard let location else {
            return nil
        }
        let distance = CLLocation(latitude: center.latitude, longitude: center.longitude).distance(
            from: CLLocation(latitude: location.lat, longitude: location.lng)
        )
        if distance < 1_000 {
            return "\(Int(distance))m"
        }
        return String(format: "%.1fkm", distance / 1_000)
    }

    private func coordinate(for location: SdzSpotLocation?) -> CLLocationCoordinate2D? {
        guard let location else {
            return nil
        }
        return CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng)
    }

    private func isParkSpot(_ spot: SdzSpot) -> Bool {
        spot.sdzIsPark
    }

    private func nearbySpotTintColor(for spot: SdzSpot) -> Color {
        isParkSpot(spot)
            ? Color.sdzPark
            : Color.sdzStreet
    }

}

final class SdzLocationSearchViewModel: NSObject, ObservableObject, MKLocalSearchCompleterDelegate {
    @Published var query: String = "" {
        didSet {
            completer.queryFragment = query
            if query.isEmpty {
                results = []
                searchItems = []
            }
            scheduleSearch()
        }
    }
    @Published var results: [MKLocalSearchCompletion] = []
    @Published var searchItems: [MKMapItem] = []

    private let completer: MKLocalSearchCompleter = MKLocalSearchCompleter()
    private let defaultRegion = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 36.2048, longitude: 138.2529),
        span: MKCoordinateSpan(latitudeDelta: 8.0, longitudeDelta: 10.0)
    )
    private var lastMapRegion: MKCoordinateRegion
    private var searchWorkItem: DispatchWorkItem?

    enum RegionUpdateSource {
        case map
        case selection
    }

    override init() {
        lastMapRegion = defaultRegion
        super.init()
        completer.delegate = self
        completer.resultTypes = [.address, .pointOfInterest]
        completer.pointOfInterestFilter = .includingAll
        completer.region = defaultRegion
    }

    var shouldShowResults: Bool {
        !query.isEmpty
    }

    var shouldShowDirectSearch: Bool {
        !query.isEmpty
    }

    func completerDidUpdateResults(_ completer: MKLocalSearchCompleter) {
        results = completer.results
    }

    func completer(_ completer: MKLocalSearchCompleter, didFailWithError error: Error) {
        results = []
    }

    func resolve(completion: MKLocalSearchCompletion, handler: @escaping (CLLocationCoordinate2D?) -> Void) {
        let request = MKLocalSearch.Request(completion: completion)
        let search = MKLocalSearch(request: request)
        search.start { response, _ in
            let coordinate = response?.mapItems.first?.location.coordinate
            DispatchQueue.main.async {
                handler(coordinate)
            }
        }
    }

    func updateRegion(_ region: MKCoordinateRegion, source: RegionUpdateSource) {
        let normalized = normalizeRegion(region)
        lastMapRegion = normalized
        completer.region = normalized
    }

    func applySelection(_ completion: MKLocalSearchCompletion) {
        query = completion.title
        results = []
        searchItems = []
    }

    func applyDirectSearchResult(name: String?) {
        if let name, !name.isEmpty {
            query = name
        }
        results = []
        searchItems = []
    }

    func reset() {
        query = ""
        results = []
        searchItems = []
    }

    func resetRegionToDefault() {
        lastMapRegion = defaultRegion
        completer.region = defaultRegion
    }

    func searchDirect(handler: @escaping (CLLocationCoordinate2D?, String?) -> Void) {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            handler(nil, nil)
            return
        }
        performDirectSearch(query: trimmed, region: lastMapRegion) { item in
            if let item {
                handler(item.location.coordinate, item.name)
                return
            }
            self.performDirectSearch(query: trimmed, region: self.defaultRegion) { fallbackItem in
                if let fallbackItem {
                    handler(fallbackItem.location.coordinate, fallbackItem.name)
                    return
                }
                let globalRegion = MKCoordinateRegion(MKMapRect.world)
                self.performDirectSearch(query: trimmed, region: globalRegion) { globalItem in
                    handler(globalItem?.location.coordinate, globalItem?.name)
                }
            }
        }
    }

    private func performDirectSearch(
        query: String,
        region: MKCoordinateRegion,
        handler: @escaping (MKMapItem?) -> Void
    ) {
        let request = MKLocalSearch.Request()
        request.naturalLanguageQuery = query
        request.resultTypes = [.address, .pointOfInterest]
        request.pointOfInterestFilter = .includingAll
        request.region = region
        let search = MKLocalSearch(request: request)
        search.start { response, _ in
            DispatchQueue.main.async {
                handler(response?.mapItems.first)
            }
        }
    }

    private func scheduleSearch() {
        searchWorkItem?.cancel()
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            searchItems = []
            return
        }
        let workItem = DispatchWorkItem { [weak self] in
            guard let self else { return }
            self.performSearchResults(query: trimmed, region: self.lastMapRegion) { items in
                self.searchItems = items
            }
        }
        searchWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.35, execute: workItem)
    }

    private func performSearchResults(
        query: String,
        region: MKCoordinateRegion,
        handler: @escaping ([MKMapItem]) -> Void
    ) {
        let request = MKLocalSearch.Request()
        request.naturalLanguageQuery = query
        request.resultTypes = [.address, .pointOfInterest]
        request.pointOfInterestFilter = .includingAll
        request.region = region
        let search = MKLocalSearch(request: request)
        search.start { response, _ in
            let items = response?.mapItems ?? []
            if !items.isEmpty {
                DispatchQueue.main.async {
                    handler(items)
                }
                return
            }
            let globalRegion = MKCoordinateRegion(MKMapRect.world)
            let fallbackRequest = MKLocalSearch.Request()
            fallbackRequest.naturalLanguageQuery = query
            fallbackRequest.resultTypes = [.address, .pointOfInterest]
            fallbackRequest.pointOfInterestFilter = .includingAll
            fallbackRequest.region = globalRegion
            let fallbackSearch = MKLocalSearch(request: fallbackRequest)
            fallbackSearch.start { fallbackResponse, _ in
                DispatchQueue.main.async {
                    handler(fallbackResponse?.mapItems ?? [])
                }
            }
        }
    }

    private func normalizeRegion(_ region: MKCoordinateRegion) -> MKCoordinateRegion {
        let minDelta: CLLocationDegrees = 0.2
        return MKCoordinateRegion(
            center: region.center,
            span: MKCoordinateSpan(
                latitudeDelta: max(region.span.latitudeDelta, minDelta),
                longitudeDelta: max(region.span.longitudeDelta, minDelta)
            )
        )
    }
}

private extension View {
    @ViewBuilder
    func onChangeCompat<T: Equatable>(of value: T, perform action: @escaping () -> Void) -> some View {
        if #available(iOS 17.0, *) {
            self.onChange(of: value) {
                action()
            }
        } else {
            self.onChange(of: value) { (_: T) in
                action()
            }
        }
    }
}

#if DEBUG
struct SdzLocationPickerView_Previews: PreviewProvider {
    static var previews: some View {
        SdzLocationPickerView(selectedLocation: .constant(SdzSpotLocation(lat: 35.6812, lng: 139.7671)))
            .environmentObject(SdzAppState())
    }
}
#endif
