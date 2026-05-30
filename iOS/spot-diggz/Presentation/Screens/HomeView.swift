import SwiftUI
import MapKit
import CoreLocation

private enum SdzSpotType: String, CaseIterable, Identifiable {
    case park
    case street

    var id: String { rawValue }

    var tagValue: String {
        switch self {
        case .park:
            return "パーク"
        case .street:
            return "ストリート"
        }
    }

    var label: String {
        switch self {
        case .park:
            return "スケートパーク"
        case .street:
            return "ストリート"
        }
    }
}

private struct TopOverlayHeightKey: PreferenceKey {
    static var defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = max(value, nextValue())
    }
}

/// Home screen showing a map with lightweight filtering.
struct HomeView: View {
    @EnvironmentObject var appState: SdzAppState
    @StateObject private var locationManager = SdzLocationManager()
    @State private var searchText: String = ""
    @State private var selectedSpotType: SdzSpotType?
    @State private var selectedTags: Set<String> = []
    @State private var spots: [SdzSpot] = []
    @State private var selectedSpot: SdzSpot?
    @State private var detailSpot: SdzSpot?
    @State private var focusedSpotId: String?
    @State private var expandedClusterSpotIds: Set<String> = []
    @State private var draftPinLocation: SdzSpotLocation?
    @State private var isLoading: Bool = false
    @State private var errorMessage: String?
    @State private var searchTask: Task<Void, Never>?
    @State private var fetchTask: Task<Void, Never>?
    @State private var topOverlayHeight: CGFloat = 0
    @State private var shouldApplySearchTextFilter: Bool = true
    @State private var showPostLocationAlert: Bool = false

    @State private var region = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 35.6812, longitude: 139.7671),
        span: MKCoordinateSpan(latitudeDelta: 0.08, longitudeDelta: 0.08)
    )
    @State private var cameraPosition: MapCameraPosition = .automatic

    var body: some View {
        Group {
            if #available(iOS 16.0, *) {
                NavigationStack {
                    homeContent
                }
            } else {
                NavigationView {
                    homeContent
                }
            }
        }
        .sheet(item: $detailSpot) { spot in
            if #available(iOS 16.0, *) {
                NavigationStack {
                    SpotDetailView(spot: spot)
                }
                .presentationDetents([.fraction(0.4), .fraction(0.9)])
                .presentationDragIndicator(.visible)
            } else {
                NavigationView {
                    SpotDetailView(spot: spot)
                }
            }
        }
    }

    private var homeContent: some View {
        GeometryReader { proxy in
            ZStack(alignment: .top) {
                mapContent(topInset: proxy.safeAreaInsets.top, topOverlayHeight: topOverlayHeight)
                    .ignoresSafeArea(edges: [.bottom, .horizontal])
                topOverlay
                floatingActionButtons(proxy: proxy)
            }
            .onPreferenceChange(TopOverlayHeightKey.self) { height in
                topOverlayHeight = height
            }
            .safeAreaInset(edge: .bottom) {
                bottomOverlay
            }
        }
        .onAppear {
            setInitialCameraPositionIfNeeded()
            fetchSpots(query: currentSearchQuery())
            applyPendingMapFocusIfNeeded()
        }
        .onReceive(locationManager.$currentCoordinate) { coordinate in
            guard let coordinate = coordinate else {
                return
            }
            let newRegion = MKCoordinateRegion(
                center: coordinate,
                span: MKCoordinateSpan(latitudeDelta: 0.02, longitudeDelta: 0.02)
            )
            withAnimation(.easeInOut(duration: 0.24)) {
                region = newRegion
                if #available(iOS 17.0, *) {
                    cameraPosition = .region(newRegion)
                }
            }
        }
        .modifier(SdzHideSpotNavigationBarModifier())
        .onChange(of: searchText) {
            scheduleSearch()
        }
        .onChange(of: selectedSpotType) {
            scheduleSearch()
        }
        .onChange(of: selectedTags) {
            scheduleSearch()
        }
        .onChange(of: appState.pendingMapFocusLocation) { _, _ in
            applyPendingMapFocusIfNeeded()
        }
        .alert("投稿位置を選択してください", isPresented: $showPostLocationAlert) {
            Button("OK", role: .cancel) {}
        } message: {
            Text("地図をタップしてピンを置いてから、右下の＋ボタンを押してください。")
        }
    }

    private struct SdzHideSpotNavigationBarModifier: ViewModifier {
        @ViewBuilder
        func body(content: Content) -> some View {
            if #available(iOS 16.0, *) {
                content.toolbar(.hidden, for: .navigationBar)
            } else {
                content.navigationBarHidden(true)
            }
        }
    }

    // MARK: - Map Content
    @ViewBuilder
    private func mapContent(topInset: CGFloat, topOverlayHeight: CGFloat) -> some View {
        SdzHomeMapView(
            annotations: annotationItems,
            region: $region,
            focusedSpotId: focusedSpotId,
            expandedSpotIds: expandedClusterSpotIds,
            onTapCoordinate: { coordinate in
                handleMapTap(coordinate)
            },
            onSelectSpot: { spot in
                selectSpot(spot)
            },
            onSelectCluster: { spotIds in
                expandedClusterSpotIds = Set(spotIds)
                selectedSpot = nil
                focusedSpotId = nil
                draftPinLocation = nil
            }
        )
    }

    private var topOverlay: some View {
        VStack(spacing: SdzSpacing.sm + 2) {
            SdzSearchBar(placeholder: "スポットを検索", text: $searchText)
            spotTypeChips
            tagFilterChips
        }
        .padding(.horizontal, SdzSpacing.lg)
        .padding(.top, SdzSpacing.md)
        .background(
            GeometryReader { proxy in
                Color.clear
                    .preference(key: TopOverlayHeightKey.self, value: proxy.size.height)
            }
        )
    }

    @ViewBuilder
    private var bottomOverlay: some View {
        VStack(spacing: SdzSpacing.md) {
            if isLoading {
                statusBanner(
                    title: "読み込み中...",
                    tone: .secondary,
                    actionTitle: nil,
                    action: nil
                )
            } else if let errorMessage = errorMessage {
                statusBanner(
                    title: errorMessage,
                    tone: .error,
                    actionTitle: "再読み込み"
                ) {
                    fetchSpots(query: currentSearchQuery())
                }
            } else if filteredSpots.isEmpty {
                statusBanner(
                    title: "スポットが見つかりません",
                    tone: .secondary,
                    actionTitle: "更新"
                ) {
                    fetchSpots(query: currentSearchQuery())
                }
            }
            if let selectedSpot = selectedSpot {
                Button {
                    detailSpot = selectedSpot
                } label: {
                    SdzSpotMiniCardView(spot: selectedSpot)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, SdzSpacing.md)
        .padding(.bottom, SdzSpacing.sm)
    }

    private enum StatusTone {
        case secondary
        case error
    }

    private func statusBanner(title: String, tone: StatusTone, actionTitle: String?, action: (() -> Void)?) -> some View {
        HStack(spacing: SdzSpacing.md) {
            Text(title)
                .font(SdzTypography.caption1)
                .foregroundColor(tone == .error ? .sdzError : .sdzTextSecondary)
            Spacer()
            if let actionTitle = actionTitle, let action = action {
                Button(actionTitle, action: action)
                    .buttonStyle(.bordered)
            }
        }
        .padding(.horizontal, SdzSpacing.md + 2)
        .padding(.vertical, SdzSpacing.sm + 2)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.md, style: .continuous))
    }

    private func floatingActionButtons(proxy: GeometryProxy) -> some View {
        VStack {
            Spacer()
            HStack {
                Spacer()
                VStack(spacing: SdzSpacing.md) {
                    Button(action: {
                        locationManager.requestCurrentLocation()
                    }) {
                        Image(systemName: "location.fill")
                            .font(.title3)
                            .padding(SdzSpacing.md)
                            .foregroundColor(.white)
                            .background(Color.sdzStreet)
                            .clipShape(Circle())
                            .sdzShadow(.md)
                    }
                    .accessibilityLabel("現在地に移動")

                    Button(action: {
                        handleTapCreateSpotButton()
                    }) {
                        Image(systemName: "plus")
                            .font(SdzTypography.title1)
                            .foregroundColor(.white)
                            .frame(width: 66, height: 66)
                            .background(
                                Circle()
                                    .fill(draftPinLocation == nil ? Color.sdzStreet : Color.sdzError)
                            )
                            .sdzShadow(.lg)
                    }
                    .accessibilityLabel("選択した位置で投稿")
                }
            }
            .padding(.trailing, SdzSpacing.lg)
            .padding(.bottom, proxy.safeAreaInsets.bottom + SdzSpacing.md)
        }
    }

    private var spotTypeChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: SdzSpacing.sm) {
                SdzChip(
                    title: "すべて",
                    isSelected: selectedSpotType == nil
                ) {
                    selectedSpotType = nil
                }
                ForEach(SdzSpotType.allCases) { spotType in
                    SdzChip(
                        title: spotType.label,
                        isSelected: selectedSpotType == spotType
                    ) {
                        selectedSpotType = spotType
                    }
                }
            }
            .padding(.vertical, SdzSpacing.xs)
        }
    }

    private var tagFilterChips: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: SdzSpacing.sm) {
                Menu {
                    ForEach(tagOptions, id: \.self) { tag in
                        Button {
                            toggleTag(tag)
                        } label: {
                            Label(tag, systemImage: selectedTags.contains(tag) ? "checkmark" : "plus")
                        }
                    }
                } label: {
                    AddTagMenuButton(isDisabled: tagOptions.isEmpty)
                }
                .buttonStyle(.plain)
                .disabled(tagOptions.isEmpty)

                ForEach(selectedTags.sorted(), id: \.self) { tag in
                    SdzChip(
                        title: tag,
                        isSelected: true,
                        systemImage: "xmark.circle.fill"
                    ) {
                        toggleTag(tag)
                    }
                }
            }
            .padding(.vertical, SdzSpacing.xs)
        }
    }

    private struct AddTagMenuButton: View {
        let isDisabled: Bool

        var body: some View {
            Image(systemName: "tag.circle.fill")
                .font(SdzTypography.title2)
                .foregroundColor(isDisabled ? .gray : Color.sdzStreet)
                .padding(.vertical, SdzSpacing.xs)
                .accessibilityLabel("タグを追加")
        }
    }

    struct AnnotationItem: Identifiable {
        let id: String
        let coordinate: CLLocationCoordinate2D
        let spot: SdzSpot?
        let isDraft: Bool
    }

    private var annotationItems: [AnnotationItem] {
        var items: [AnnotationItem] = filteredSpots.compactMap { spot in
            if let location = spot.location {
                return AnnotationItem(
                    id: spot.spotId,
                    coordinate: CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng),
                    spot: spot,
                    isDraft: false
                )
            }
            return nil
        }
        if let draftPinLocation = draftPinLocation {
            items.append(
                AnnotationItem(
                    id: "draft-pin",
                    coordinate: CLLocationCoordinate2D(latitude: draftPinLocation.lat, longitude: draftPinLocation.lng),
                    spot: nil,
                    isDraft: true
                )
            )
        }
        return items
    }

    private var tagOptions: [String] {
        let typeTags = Set(SdzSpotType.allCases.map(\.tagValue))
        let tags = spots
            .flatMap(\.tags)
            .filter { !$0.isEmpty && !typeTags.contains($0) }
        return Array(Set(tags)).sorted()
    }

    /// Filters spots based on the search text and selected tag.
    private var filteredSpots: [SdzSpot] {
        spots
    }

    private func toggleTag(_ tag: String) {
        if selectedTags.contains(tag) {
            selectedTags.remove(tag)
        } else {
            selectedTags.insert(tag)
        }
    }

    private func selectSpot(_ spot: SdzSpot) {
        focusedSpotId = spot.spotId
        draftPinLocation = nil
        selectedSpot = spot
        expandedClusterSpotIds = []
    }

    private func focusOnSpot(_ spot: SdzSpot) {
        guard let location = spot.location else {
            return
        }
        let coordinate = CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng)
        focusOnCoordinate(coordinate)
    }

    private func pinColor(for spot: SdzSpot) -> Color {
        spot.sdzPinColor
    }

    private func isParkSpot(_ spot: SdzSpot) -> Bool {
        spot.sdzIsPark
    }

    private func focusOnCoordinate(_ coordinate: CLLocationCoordinate2D) {
        let newRegion = MKCoordinateRegion(
            center: coordinate,
            span: MKCoordinateSpan(latitudeDelta: 0.02, longitudeDelta: 0.02)
        )
        region = newRegion
        if #available(iOS 17.0, *) {
            cameraPosition = .region(newRegion)
        }
    }

    private func applyPendingMapFocusIfNeeded() {
        guard let location = appState.pendingMapFocusLocation else {
            return
        }
        draftPinLocation = location
        focusOnCoordinate(CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng))
        appState.pendingMapFocusLocation = nil
        appState.pendingMapFocusName = nil
    }

    private func setInitialCameraPositionIfNeeded() {
        if #available(iOS 17.0, *) {
            if case .automatic = cameraPosition {
                cameraPosition = .region(region)
            }
        }
    }

    private func handleMapTap(_ coordinate: CLLocationCoordinate2D) {
        if let spot = nearestSpot(to: coordinate, within: 60) {
            selectSpot(spot)
            return
        }
        selectedSpot = nil
        focusedSpotId = nil
        expandedClusterSpotIds = []
        draftPinLocation = SdzSpotLocation(lat: coordinate.latitude, lng: coordinate.longitude)
    }

    private func nearestSpot(to coordinate: CLLocationCoordinate2D, within meters: CLLocationDistance) -> SdzSpot? {
        let tapLocation = CLLocation(latitude: coordinate.latitude, longitude: coordinate.longitude)
        return filteredSpots.first { spot in
            guard let location = spot.location else {
                return false
            }
            let spotLocation = CLLocation(latitude: location.lat, longitude: location.lng)
            return tapLocation.distance(from: spotLocation) <= meters
        }
    }

    private func openPostForDraftPin(_ location: SdzSpotLocation) {
        appState.draftPostLocation = location
        appState.selectedTab = .spots
        appState.isPostComposerPresented = true
        draftPinLocation = nil
    }

    private func handleTapCreateSpotButton() {
        guard let draftPinLocation else {
            showPostLocationAlert = true
            return
        }
        openPostForDraftPin(draftPinLocation)
    }

    /// Loads the list of spots from the API.
    private func fetchSpots(query: SdzSpotSearchQuery?) {
        fetchTask?.cancel()
        isLoading = true
        errorMessage = nil
        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
        fetchTask = Task {
            do {
                var result = try await apiClient.fetchSpots(query: query, includeAuth: true)
                #if DEBUG
                if !result.contains(where: { $0.approvalStatus == .approved }) {
                    result.append(
                        SdzSpot(
                            spotId: "debug-approved-park",
                            name: "承認済みパーク（デバッグ）",
                            description: nil,
                            location: SdzSpotLocation(lat: 35.6812, lng: 139.7671),
                            tags: ["パーク"],
                            images: [],
                            approvalStatus: .approved,
                            parkAttributes: SdzSpotParkAttributes(
                                officialUrl: nil,
                                businessHours: nil,
                                accessInfo: nil,
                                phoneNumber: nil
                            ),
                            streetAttributes: nil,
                            userId: "debug",
                            createdAt: Date(),
                            updatedAt: Date()
                        )
                    )
                }
                if !result.contains(where: { $0.approvalStatus != .approved }) {
                    result.append(
                        SdzSpot(
                            spotId: "debug-pending-street",
                            name: "未承認ストリート（デバッグ）",
                            description: nil,
                            location: SdzSpotLocation(lat: 35.6762, lng: 139.7649),
                            tags: ["ストリート"],
                            images: [],
                            approvalStatus: .pending,
                            parkAttributes: nil,
                            streetAttributes: SdzStreetAttributes(
                                surfaceMaterial: nil,
                                surfaceCondition: nil,
                                sections: nil,
                                difficulty: nil,
                                notes: nil
                            ),
                            userId: "debug",
                            createdAt: Date(),
                            updatedAt: Date()
                        )
                    )
                }
                #endif
                await MainActor.run {
                    self.spots = result
                    let validSpotIds = Set(result.map(\.spotId))
                    self.expandedClusterSpotIds = self.expandedClusterSpotIds.intersection(validSpotIds)
                    self.isLoading = false
                }
            } catch {
                let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
                await MainActor.run {
                    self.errorMessage = message
                    self.isLoading = false
                }
            }
        }
    }

    private func scheduleSearch() {
        searchTask?.cancel()
        let trimmedSearchText = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        let searchRegion = region
        searchTask = Task {
            try? await Task.sleep(nanoseconds: 350_000_000)
            guard !Task.isCancelled else {
                return
            }
            var searchedCoordinate: CLLocationCoordinate2D?
            if !trimmedSearchText.isEmpty {
                searchedCoordinate = await resolveSearchCoordinate(
                    keyword: trimmedSearchText,
                    preferredRegion: searchRegion
                )
            }
            guard !Task.isCancelled else {
                return
            }
            await MainActor.run {
                if trimmedSearchText.isEmpty {
                    shouldApplySearchTextFilter = true
                } else if let searchedCoordinate {
                    shouldApplySearchTextFilter = false
                    selectedSpot = nil
                    focusedSpotId = nil
                    focusOnCoordinate(searchedCoordinate)
                } else {
                    shouldApplySearchTextFilter = true
                }
                fetchSpots(query: currentSearchQuery())
            }
        }
    }

    private func currentSearchQuery() -> SdzSpotSearchQuery? {
        let query = SdzSpotSearchQuery(
            text: shouldApplySearchTextFilter ? searchText : nil,
            spotType: selectedSpotType?.rawValue,
            tags: Array(selectedTags)
        )
        return query.isEmpty ? nil : query
    }

    private func resolveSearchCoordinate(
        keyword: String,
        preferredRegion: MKCoordinateRegion
    ) async -> CLLocationCoordinate2D? {
        if let coordinate = await performLocationSearch(
            keyword: keyword,
            region: preferredRegion
        ) {
            return coordinate
        }
        let worldRegion = MKCoordinateRegion(MKMapRect.world)
        return await performLocationSearch(
            keyword: keyword,
            region: worldRegion
        )
    }

    private func performLocationSearch(
        keyword: String,
        region: MKCoordinateRegion
    ) async -> CLLocationCoordinate2D? {
        await withCheckedContinuation { continuation in
            let request = MKLocalSearch.Request()
            request.naturalLanguageQuery = keyword
            request.resultTypes = [.address, .pointOfInterest]
            request.pointOfInterestFilter = .includingAll
            request.region = region
            let search = MKLocalSearch(request: request)
            search.start { response, _ in
                continuation.resume(returning: response?.mapItems.first?.placemark.coordinate)
            }
        }
    }
}

private struct SdzHomeMapView: UIViewRepresentable {
    let annotations: [HomeView.AnnotationItem]
    @Binding var region: MKCoordinateRegion
    let focusedSpotId: String?
    let expandedSpotIds: Set<String>
    let onTapCoordinate: (CLLocationCoordinate2D) -> Void
    let onSelectSpot: (SdzSpot) -> Void
    let onSelectCluster: ([String]) -> Void

    func makeUIView(context: Context) -> MKMapView {
        let mapView = MKMapView(frame: .zero)
        mapView.delegate = context.coordinator
        mapView.showsCompass = false
        mapView.showsUserLocation = true
        mapView.setRegion(region, animated: false)
        let tap = UITapGestureRecognizer(target: context.coordinator, action: #selector(Coordinator.handleMapTap(_:)))
        tap.cancelsTouchesInView = false
        tap.delegate = context.coordinator
        mapView.addGestureRecognizer(tap)
        return mapView
    }

    func updateUIView(_ mapView: MKMapView, context: Context) {
        if !context.coordinator.isRegionChangeFromMap {
            if !context.coordinator.isSimilarRegion(lhs: mapView.region, rhs: region) {
                mapView.setRegion(region, animated: true)
            }
        }
        context.coordinator.isRegionChangeFromMap = false
        context.coordinator.focusedSpotId = focusedSpotId
        context.coordinator.expandedSpotIds = expandedSpotIds
        context.coordinator.updateAnnotations(annotations, in: mapView)
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    final class Coordinator: NSObject, MKMapViewDelegate, UIGestureRecognizerDelegate {
        private let parent: SdzHomeMapView
        private var annotationStore: [String: SdzSpotAnnotation] = [:]
        private var clusterStore: [String: SdzClusterAnnotation] = [:]
        fileprivate var focusedSpotId: String?
        fileprivate var expandedSpotIds: Set<String> = []
        fileprivate var isRegionChangeFromMap = false
        private var lastFocusedSpotId: String?
        private var isRegionChanging = false

        init(parent: SdzHomeMapView) {
            self.parent = parent
            self.focusedSpotId = parent.focusedSpotId
            self.expandedSpotIds = parent.expandedSpotIds
        }

        func updateAnnotations(_ items: [HomeView.AnnotationItem], in mapView: MKMapView) {
            let previousFocusId = lastFocusedSpotId
            let currentFocusId = focusedSpotId
            lastFocusedSpotId = currentFocusId
            let region = parent.region
            let gridSizePoints = clusterGridSizePoints(for: region)

            var nextIds = Set<String>()
            for item in items {
                nextIds.insert(item.id)
                if let existing = annotationStore[item.id] {
                    existing.coordinate = item.coordinate
                    existing.spot = item.spot
                    existing.isDraft = item.isDraft
                } else {
                    let annotation = SdzSpotAnnotation(
                        id: item.id,
                        coordinate: item.coordinate,
                        spot: item.spot,
                        isDraft: item.isDraft
                    )
                    annotationStore[item.id] = annotation
                }
            }
            let removed = annotationStore.keys.filter { !nextIds.contains($0) }
            for id in removed {
                if let annotation = annotationStore.removeValue(forKey: id) {
                    mapView.removeAnnotation(annotation)
                }
            }

            var displaySpotIds = Set<String>()
            var displayClusterIds = Set<String>()
            var clusterGroups: [ClusterKey: [SdzSpotAnnotation]] = [:]

            for item in items where !item.isDraft {
                guard let annotation = annotationStore[item.id], let spot = item.spot else { continue }
                if item.id == focusedSpotId || expandedSpotIds.contains(item.id) {
                    displaySpotIds.insert(item.id)
                    continue
                }
                let mapPoint = MKMapPoint(item.coordinate)
                let key = ClusterKey(
                    type: clusterTypeKey(for: spot),
                    x: Int(floor(mapPoint.x / gridSizePoints)),
                    y: Int(floor(mapPoint.y / gridSizePoints))
                )
                clusterGroups[key, default: []].append(annotation)
            }

            for (key, annotations) in clusterGroups {
                if annotations.count == 1, let annotation = annotations.first {
                    displaySpotIds.insert(annotation.id)
                } else if !annotations.isEmpty {
                    let clusterId = "\(key.type)-\(key.x)-\(key.y)"
                    let cluster = clusterStore[clusterId] ?? SdzClusterAnnotation(
                        id: clusterId,
                        coordinate: annotations[0].coordinate,
                        count: annotations.count,
                        isPark: key.type == "park",
                        memberMapRect: MKMapRect.null,
                        memberSpotIds: []
                    )
                    updateCluster(cluster, with: annotations)
                    clusterStore[clusterId] = cluster
                    displayClusterIds.insert(clusterId)
                }
            }

            let removedClusters = clusterStore.keys.filter { !displayClusterIds.contains($0) }
            for id in removedClusters {
                if let cluster = clusterStore.removeValue(forKey: id) {
                    mapView.removeAnnotation(cluster)
                }
            }

            for (id, annotation) in annotationStore {
                let shouldDisplay = annotation.isDraft || displaySpotIds.contains(id)
                setAnnotation(annotation, visible: shouldDisplay, in: mapView)
            }

            for (id, cluster) in clusterStore {
                let shouldDisplay = displayClusterIds.contains(id)
                setAnnotation(cluster, visible: shouldDisplay, in: mapView)
            }

            mapView.annotations.forEach { annotation in
                if let view = mapView.view(for: annotation) as? SdzSpotAnnotationView,
                   let spotAnnotation = annotation as? SdzSpotAnnotation,
                   let spot = spotAnnotation.spot {
                    view.update(spot: spot, isFocused: focusedSpotId == spot.spotId)
                }
            }

            if previousFocusId != currentFocusId, !isRegionChanging {
                let idsToRefresh = [previousFocusId, currentFocusId].compactMap { $0 }
                for id in idsToRefresh {
                    refreshAnnotation(id: id, in: mapView)
                }
            } else if previousFocusId != currentFocusId {
                let idsToRefresh = [previousFocusId, currentFocusId].compactMap { $0 }
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self, weak mapView] in
                    guard let self, let mapView else { return }
                    for id in idsToRefresh {
                        self.refreshAnnotation(id: id, in: mapView)
                    }
                }
            }
        }

        @objc func handleMapTap(_ gesture: UITapGestureRecognizer) {
            guard let mapView = gesture.view as? MKMapView else { return }
            let point = gesture.location(in: mapView)
            let coordinate = mapView.convert(point, toCoordinateFrom: mapView)
            parent.onTapCoordinate(coordinate)
        }

        func mapViewDidChangeVisibleRegion(_ mapView: MKMapView) {
            isRegionChangeFromMap = true
            parent.region = mapView.region
        }

        func mapView(_ mapView: MKMapView, regionWillChangeAnimated animated: Bool) {
            isRegionChanging = true
        }

        func mapView(_ mapView: MKMapView, regionDidChangeAnimated animated: Bool) {
            isRegionChanging = false
        }

        func mapView(_ mapView: MKMapView, viewFor annotation: MKAnnotation) -> MKAnnotationView? {
            if annotation is MKUserLocation {
                return nil
            }
            if let cluster = annotation as? SdzClusterAnnotation {
                let identifier = "SdzClusterAnnotationView"
                let view = mapView.dequeueReusableAnnotationView(withIdentifier: identifier) as? SdzClusterAnnotationView
                    ?? SdzClusterAnnotationView(annotation: cluster, reuseIdentifier: identifier)
                view.annotation = cluster
                view.canShowCallout = false
                view.update(
                    count: cluster.count,
                    tintColor: clusterTintColor(for: cluster)
                )
                view.displayPriority = .required
                return view
            }
            guard let spotAnnotation = annotation as? SdzSpotAnnotation else {
                return nil
            }
            if spotAnnotation.isDraft {
                let identifier = "SdzDraftAnnotationView"
                let view = mapView.dequeueReusableAnnotationView(withIdentifier: identifier) ?? MKAnnotationView(annotation: spotAnnotation, reuseIdentifier: identifier)
                view.annotation = spotAnnotation
                view.image = UIImage(systemName: "plus.circle.fill")
                view.displayPriority = .required
                view.clusteringIdentifier = nil
                return view
            }
            let identifier = "SdzSpotAnnotationView"
            let view = mapView.dequeueReusableAnnotationView(withIdentifier: identifier) as? SdzSpotAnnotationView
                ?? SdzSpotAnnotationView(annotation: spotAnnotation, reuseIdentifier: identifier)
            view.annotation = spotAnnotation
            if let spot = spotAnnotation.spot {
                view.update(spot: spot, isFocused: focusedSpotId == spot.spotId)
            }
            view.clusteringIdentifier = nil
            view.displayPriority = .required
            return view
        }

        func mapView(_ mapView: MKMapView, didSelect view: MKAnnotationView) {
            if let cluster = view.annotation as? SdzClusterAnnotation {
                parent.onSelectCluster(cluster.memberSpotIds)
                zoomToCluster(cluster, in: mapView)
                return
            }
            guard let annotation = view.annotation as? SdzSpotAnnotation,
                  let spot = annotation.spot,
                  !annotation.isDraft else { return }
            focusedSpotId = spot.spotId
            if let spotView = view as? SdzSpotAnnotationView {
                spotView.update(spot: spot, isFocused: true)
            }
            parent.onSelectSpot(spot)
        }

        func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldReceive touch: UITouch) -> Bool {
            if let view = touch.view, isAnnotationSubview(view) {
                return false
            }
            return true
        }

        private func isAnnotationSubview(_ view: UIView) -> Bool {
            var current: UIView? = view
            while let node = current {
                if node is MKAnnotationView {
                    return true
                }
                current = node.superview
            }
            return false
        }

        private func clusterTintColor(for cluster: SdzClusterAnnotation) -> UIColor {
            UIColor(cluster.isPark ? Color.sdzPark : Color.sdzStreet)
        }

        private func zoomToCluster(_ cluster: SdzClusterAnnotation, in mapView: MKMapView) {
            guard !cluster.memberMapRect.isNull else { return }
            let padding = UIEdgeInsets(top: 80, left: 80, bottom: 80, right: 80)
            mapView.setVisibleMapRect(cluster.memberMapRect, edgePadding: padding, animated: true)
        }

        private func refreshAnnotation(id: String, in mapView: MKMapView) {
            guard let annotation = annotationStore[id] else { return }
            mapView.removeAnnotation(annotation)
            mapView.addAnnotation(annotation)
        }

        private func setAnnotation(_ annotation: MKAnnotation, visible: Bool, in mapView: MKMapView) {
            let exists = mapView.annotations.contains { $0 === annotation }
            if visible && !exists {
                mapView.addAnnotation(annotation)
            } else if !visible && exists {
                mapView.removeAnnotation(annotation)
            }
        }

        private func clusterGridSizePoints(for region: MKCoordinateRegion) -> Double {
            let radiusMeters = clusterRadiusMeters(for: region.span.latitudeDelta)
            let metersPerPoint = MKMetersPerMapPointAtLatitude(region.center.latitude)
            return max(1, radiusMeters / metersPerPoint)
        }

        private func clusterRadiusMeters(for latitudeDelta: CLLocationDegrees) -> Double {
            switch latitudeDelta {
            case ..<0.05:
                return 500
            case ..<0.5:
                return 5_000
            case ..<2.0:
                return 10_000
            case ..<8.0:
                return 100_000
            default:
                return 300_000
            }
        }

        private func clusterTypeKey(for spot: SdzSpot) -> String {
            spot.sdzIsPark ? "park" : "street"
        }

        private func updateCluster(_ cluster: SdzClusterAnnotation, with annotations: [SdzSpotAnnotation]) {
            guard !annotations.isEmpty else { return }
            var rect = MKMapRect.null
            var sumX = 0.0
            var sumY = 0.0
            for annotation in annotations {
                let point = MKMapPoint(annotation.coordinate)
                sumX += point.x
                sumY += point.y
                let pointRect = MKMapRect(x: point.x, y: point.y, width: 0.01, height: 0.01)
                rect = rect.isNull ? pointRect : rect.union(pointRect)
            }
            let count = Double(annotations.count)
            let centerPoint = MKMapPoint(x: sumX / count, y: sumY / count)
            cluster.coordinate = centerPoint.coordinate
            cluster.count = annotations.count
            if let spot = annotations.first?.spot {
                cluster.isPark = spot.sdzIsPark
            }
            cluster.memberMapRect = rect
            cluster.memberSpotIds = annotations.compactMap { $0.spot?.spotId }
        }

        private struct ClusterKey: Hashable {
            let type: String
            let x: Int
            let y: Int
        }

        fileprivate func isSimilarRegion(lhs: MKCoordinateRegion, rhs: MKCoordinateRegion) -> Bool {
            let delta = abs(lhs.center.latitude - rhs.center.latitude)
                + abs(lhs.center.longitude - rhs.center.longitude)
                + abs(lhs.span.latitudeDelta - rhs.span.latitudeDelta)
                + abs(lhs.span.longitudeDelta - rhs.span.longitudeDelta)
            return delta < 0.0001
        }
    }
}

private final class SdzSpotAnnotation: NSObject, MKAnnotation {
    let id: String
    dynamic var coordinate: CLLocationCoordinate2D
    var spot: SdzSpot?
    var isDraft: Bool

    init(id: String, coordinate: CLLocationCoordinate2D, spot: SdzSpot?, isDraft: Bool) {
        self.id = id
        self.coordinate = coordinate
        self.spot = spot
        self.isDraft = isDraft
    }
}

private final class SdzClusterAnnotation: NSObject, MKAnnotation {
    let id: String
    dynamic var coordinate: CLLocationCoordinate2D
    var count: Int
    var isPark: Bool
    var memberMapRect: MKMapRect
    var memberSpotIds: [String]

    init(
        id: String,
        coordinate: CLLocationCoordinate2D,
        count: Int,
        isPark: Bool,
        memberMapRect: MKMapRect,
        memberSpotIds: [String]
    ) {
        self.id = id
        self.coordinate = coordinate
        self.count = count
        self.isPark = isPark
        self.memberMapRect = memberMapRect
        self.memberSpotIds = memberSpotIds
    }
}

private final class SdzSpotAnnotationView: MKAnnotationView {
    private var hostingController: UIHostingController<AnyView>?

    override init(annotation: MKAnnotation?, reuseIdentifier: String?) {
        super.init(annotation: annotation, reuseIdentifier: reuseIdentifier)
        setup()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setup()
    }

    private func setup() {
        canShowCallout = false
        backgroundColor = .clear
    }

    func update(spot: SdzSpot, isFocused: Bool) {
        let view = AnyView(
            SdzMapPinView(spot: spot, isFocused: isFocused)
        )
        if let hostingController = hostingController {
            hostingController.rootView = view
        } else {
            let controller = UIHostingController(rootView: view)
            controller.view.backgroundColor = .clear
            controller.view.translatesAutoresizingMaskIntoConstraints = true
            hostingController = controller
            addSubview(controller.view)
        }
        layoutHostingView()
    }

    private func layoutHostingView() {
        guard let hostingController = hostingController else { return }
        let hostedView = hostingController.view!
        let size = hostingController.sizeThatFits(in: CGSize(width: 240, height: 240))
        frame = CGRect(origin: .zero, size: size)
        hostedView.frame = bounds
        centerOffset = CGPoint(x: 0, y: -size.height * 0.5)
    }
}

private final class SdzClusterAnnotationView: MKAnnotationView {
    private var hostingController: UIHostingController<AnyView>?

    override init(annotation: MKAnnotation?, reuseIdentifier: String?) {
        super.init(annotation: annotation, reuseIdentifier: reuseIdentifier)
        canShowCallout = false
        backgroundColor = .clear
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        canShowCallout = false
        backgroundColor = .clear
    }

    func update(count: Int, tintColor: UIColor) {
        let view = AnyView(
            SdzClusterBubble(count: count, color: Color(tintColor))
        )
        if let hostingController = hostingController {
            hostingController.rootView = view
        } else {
            let controller = UIHostingController(rootView: view)
            controller.view.backgroundColor = .clear
            controller.view.translatesAutoresizingMaskIntoConstraints = true
            hostingController = controller
            addSubview(controller.view)
        }
        layoutHostingView()
    }

    private func layoutHostingView() {
        guard let hostingController = hostingController else { return }
        let hostedView = hostingController.view!
        let size = hostingController.sizeThatFits(in: CGSize(width: 120, height: 120))
        frame = CGRect(origin: .zero, size: size)
        hostedView.frame = bounds
        centerOffset = CGPoint(x: 0, y: -size.height * 0.5)
    }
}


#if DEBUG
struct HomeView_Previews: PreviewProvider {
    static var previews: some View {
        HomeView()
            .environmentObject(SdzAppState())
    }
}
#endif
