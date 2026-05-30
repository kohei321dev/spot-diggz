import Foundation
import Combine

enum SdzTab: Hashable {
    case spots
    case list
    case settings
}

/// Shared application state used across the entire app.
@MainActor
final class SdzAppState: ObservableObject {
    /// Flag indicating whether the user is authenticated.
    @Published var isAuthenticated: Bool = false

    /// Currently selected API environment.
    let environment: SdzEnvironment = .dev

    /// Placeholder for the authentication token (Firebase ID token).
    @Published var idToken: String?

    /// Current authenticated user info (from Firebase).
    @Published var authUserId: String?
    @Published var authEmail: String?
    @Published var authDisplayName: String?
    @Published var authProviderIds: [String] = []

    /// Indicates whether the auth session is loading.
    @Published var isAuthLoading: Bool = false

    /// Favorites synced with backend (cached locally).
    @Published private(set) var favoriteSpots: [SdzSpot] = []
    @Published var isFavoritesLoading: Bool = false
    @Published var favoritesErrorMessage: String?

    /// Profile image data stored locally.
    @Published var profileImageData: Data?

    /// Currently selected tab.
    @Published var selectedTab: SdzTab = .spots
    @Published var isPostComposerPresented: Bool = false

    /// Indicates whether the post or edit screen is currently visible.
    @Published var isPostingSpot: Bool = false
    @Published var isEditingSpot: Bool = false

    /// Draft location passed from map to the post flow.
    @Published var draftPostLocation: SdzSpotLocation?

    /// Pending official URL shared from external apps.
    @Published var pendingOfficialUrl: String?

    /// Pending spot location shared from external map apps.
    @Published var pendingSharedLocation: SdzSpotLocation?
    @Published var pendingSharedLocationName: String?
    @Published var pendingSharedLocationError: String?
    @Published var pendingSharedLocationForEdit: SdzSpotLocation?
    @Published var pendingSharedLocationNameForEdit: String?
    @Published var pendingSharedLocationErrorForEdit: String?

    /// Pending location to focus on the map when sharing from external apps.
    @Published var pendingMapFocusLocation: SdzSpotLocation?
    @Published var pendingMapFocusName: String?

    /// Pending selection when no return context is available.
    @Published var pendingShareSelectionLocation: SdzSpotLocation?
    @Published var pendingShareSelectionName: String?
    @Published var isShareSelectionPromptVisible: Bool = false

    private enum SharedDefaults {
        static let appGroupId = "group.ios-sdz-fb-dev"
        static let payloadKey = "sdz.shared-payload"
    }
    private static let universalLinkHost = "sdz-dev-api-1053202159855.asia-northeast1.run.app"

    enum SdzShareReturnContext: String, Codable {
        case post
        case edit
        case map
    }

    private enum ShareReturnContextDefaults {
        static let key = "sdz.share-return-context"
        static let timestampKey = "sdz.share-return-context-date"
        static let maxAge: TimeInterval = 15 * 60
    }

    private struct SharedPayload: Codable {
        let kind: String
        let lat: Double?
        let lng: Double?
        let name: String?
        let url: String?
        let createdAt: Date
    }

    init() {
        loadFavorites()
        loadProfileImage()
    }

    func setShareReturnContext(_ context: SdzShareReturnContext?) {
        if let context = context {
            UserDefaults.standard.set(context.rawValue, forKey: ShareReturnContextDefaults.key)
            UserDefaults.standard.set(Date().timeIntervalSince1970, forKey: ShareReturnContextDefaults.timestampKey)
        } else {
            clearShareReturnContext()
        }
    }

    func consumeSharedPayloadIfNeeded() {
        guard let defaults = UserDefaults(suiteName: SharedDefaults.appGroupId),
              let data = defaults.data(forKey: SharedDefaults.payloadKey) else {
            return
        }
        defaults.removeObject(forKey: SharedDefaults.payloadKey)
        guard let payload = try? JSONDecoder().decode(SharedPayload.self, from: data) else {
            return
        }
        pendingSharedLocationError = nil
        pendingSharedLocationErrorForEdit = nil
        switch payload.kind {
        case "officialUrl":
            if let url = payload.url, !url.isEmpty {
                pendingOfficialUrl = url
                selectedTab = .spots
                isPostComposerPresented = true
            }
        case "location":
            let context = resolveShareReturnContext()
            if let lat = payload.lat, let lng = payload.lng {
                applySharedLocation(
                    location: SdzSpotLocation(lat: lat, lng: lng),
                    name: payload.name,
                    context: context
                )
                return
            }
            if let url = payload.url, !url.isEmpty {
                Task {
                    await handleSharedLocation(urlString: url, context: context)
                }
            }
        default:
            break
        }
    }

    func applyShareSelectionToPost() {
        guard let location = pendingShareSelectionLocation else {
            return
        }
        pendingSharedLocation = location
        pendingSharedLocationName = pendingShareSelectionName
        pendingShareSelectionLocation = nil
        pendingShareSelectionName = nil
        isShareSelectionPromptVisible = false
        selectedTab = .spots
        isPostComposerPresented = true
    }

    func applyShareSelectionToMap() {
        guard let location = pendingShareSelectionLocation else {
            return
        }
        pendingMapFocusLocation = location
        pendingMapFocusName = pendingShareSelectionName
        pendingShareSelectionLocation = nil
        pendingShareSelectionName = nil
        isShareSelectionPromptVisible = false
        selectedTab = .spots
    }

    func clearShareSelection() {
        pendingShareSelectionLocation = nil
        pendingShareSelectionName = nil
        isShareSelectionPromptVisible = false
    }

    func restoreSession() async {
        isAuthLoading = true
        let session = await SdzAuthService.shared.currentSession()
        applySession(session)
        isAuthLoading = false
    }

    func signInWithEmail(email: String, password: String) async throws {
        let session = try await SdzAuthService.shared.signInWithEmail(email: email, password: password)
        applySession(session)
    }

    func signUpWithEmail(email: String, password: String) async throws {
        let session = try await SdzAuthService.shared.signUpWithEmail(email: email, password: password)
        applySession(session)
    }

    func signInWithGoogle() async throws {
        let session = try await SdzAuthService.shared.signInWithGoogle()
        applySession(session)
    }

    func signOut() {
        do {
            try SdzAuthService.shared.signOut()
        } catch {
            // Keep state consistent even if sign-out fails.
        }
        applySession(nil)
    }

    func handleIncomingUrl(_ url: URL) -> Bool {
        if url.scheme == "sdz" {
            return handleShareUrl(host: url.host, path: nil, url: url)
        }
        if url.scheme == "https" {
            guard let host = url.host,
                  host == Self.universalLinkHost else {
                return false
            }
            return handleShareUrl(host: nil, path: url.path, url: url)
        }
        return false
    }

    private func handleShareUrl(host: String?, path: String?, url: URL) -> Bool {
        let target = host ?? path
        guard let target else {
            return false
        }
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
            return false
        }
        let context = resolveShareReturnContext()

        if target == "add-url" || target.hasPrefix("/add-url") {
            guard let value = components.queryItems?.first(where: { $0.name == "url" })?.value,
                  !value.isEmpty else {
                return false
            }
            pendingOfficialUrl = value
            selectedTab = .spots
            isPostComposerPresented = true
            clearSharedPayload()
            return true
        }
        if target == "share-location" || target.hasPrefix("/share-location") {
            if let latString = components.queryItems?.first(where: { $0.name == "lat" })?.value,
               let lngString = components.queryItems?.first(where: { $0.name == "lng" })?.value,
               let lat = Double(latString),
               let lng = Double(lngString) {
                applySharedLocation(
                    location: SdzSpotLocation(lat: lat, lng: lng),
                    name: components.queryItems?.first(where: { $0.name == "name" })?.value,
                    context: context
                )
                clearSharedPayload()
                return true
            }
            guard let value = components.queryItems?.first(where: { $0.name == "url" })?.value,
                  !value.isEmpty else {
                return false
            }
            Task {
                await handleSharedLocation(urlString: value, context: context)
                clearSharedPayload()
            }
            return true
        }
        return false
    }

    private func handleSharedLocation(urlString: String, context: SdzShareReturnContext?) async {
        pendingSharedLocationError = nil
        if let payload = parseSharedLocation(urlString: urlString) {
            applySharedLocation(location: payload.location, name: payload.name, context: context)
            return
        }
        if let resolved = await resolveRedirectUrl(urlString: urlString),
           let payload = parseSharedLocation(urlString: resolved.absoluteString) {
            applySharedLocation(location: payload.location, name: payload.name, context: context)
            return
        }
        applySharedLocationError(context: context)
    }

    private func applySharedLocation(
        location: SdzSpotLocation,
        name: String?,
        context: SdzShareReturnContext?
    ) {
        switch context {
        case .map:
            pendingMapFocusLocation = location
            pendingMapFocusName = name
            selectedTab = .spots
        case .edit:
            pendingSharedLocationForEdit = location
            pendingSharedLocationNameForEdit = name
        case .post:
            pendingSharedLocation = location
            pendingSharedLocationName = name
            selectedTab = .spots
            isPostComposerPresented = true
        case .none:
            pendingShareSelectionLocation = location
            pendingShareSelectionName = name
            isShareSelectionPromptVisible = true
        }
    }

    private func applySharedLocationError(context: SdzShareReturnContext?) {
        let message = "位置情報を取得できませんでした。Appleマップで場所を開いたまま共有してください。"
        switch context {
        case .edit:
            pendingSharedLocationErrorForEdit = message
        case .map, .post:
            pendingSharedLocationError = message
        case .none:
            pendingSharedLocationError = message
        }
    }

    private func resolveShareReturnContext() -> SdzShareReturnContext? {
        guard let raw = UserDefaults.standard.string(forKey: ShareReturnContextDefaults.key),
              let context = SdzShareReturnContext(rawValue: raw) else {
            return nil
        }
        let timestamp = UserDefaults.standard.double(forKey: ShareReturnContextDefaults.timestampKey)
        if timestamp > 0 {
            let age = Date().timeIntervalSince1970 - timestamp
            if age > ShareReturnContextDefaults.maxAge {
                clearShareReturnContext()
                return nil
            }
        }
        if context == .edit, !isEditingSpot {
            clearShareReturnContext()
            return nil
        }
        if context == .post, !isPostingSpot {
            clearShareReturnContext()
            return nil
        }
        clearShareReturnContext()
        return context
    }

    private func clearShareReturnContext() {
        UserDefaults.standard.removeObject(forKey: ShareReturnContextDefaults.key)
        UserDefaults.standard.removeObject(forKey: ShareReturnContextDefaults.timestampKey)
    }

    private func clearSharedPayload() {
        guard let defaults = UserDefaults(suiteName: SharedDefaults.appGroupId) else {
            return
        }
        defaults.removeObject(forKey: SharedDefaults.payloadKey)
    }

    private struct SdzSharedLocationPayload {
        let location: SdzSpotLocation
        let name: String?
    }

    private func parseSharedLocation(urlString: String) -> SdzSharedLocationPayload? {
        guard let url = URL(string: urlString) else {
            return nil
        }
        if let payload = parseAppleMaps(url: url) {
            return payload
        }
        if let payload = parseGoogleMaps(url: url) {
            return payload
        }
        return nil
    }

    private func parseAppleMaps(url: URL) -> SdzSharedLocationPayload? {
        guard let host = url.host?.lowercased(), host.contains("maps.apple.com") else {
            return nil
        }
        let components = URLComponents(url: url, resolvingAgainstBaseURL: false)
        let items = components?.queryItems ?? []
        let name = value(for: "q", in: items) ?? value(for: "name", in: items) ?? value(for: "address", in: items)
        if let ll = value(for: "ll", in: items), let location = parseCoordinatePair(from: ll) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        if let sll = value(for: "sll", in: items), let location = parseCoordinatePair(from: sll) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        if let center = value(for: "center", in: items), let location = parseCoordinatePair(from: center) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        return nil
    }

    private func parseGoogleMaps(url: URL) -> SdzSharedLocationPayload? {
        guard let host = url.host?.lowercased() else {
            return nil
        }
        guard host.contains("google.com") || host.contains("maps.google.com") || host.contains("maps.app.goo.gl") || host.contains("goo.gl") else {
            return nil
        }
        let components = URLComponents(url: url, resolvingAgainstBaseURL: false)
        let items = components?.queryItems ?? []
        let name = value(for: "q", in: items) ?? value(for: "query", in: items)
        if let q = value(for: "q", in: items), let location = parseCoordinatePair(from: q) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        if let query = value(for: "query", in: items), let location = parseCoordinatePair(from: query) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        if let ll = value(for: "ll", in: items), let location = parseCoordinatePair(from: ll) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        if let location = parseCoordinateFromPath(url.absoluteString) {
            return SdzSharedLocationPayload(location: location, name: name)
        }
        return nil
    }

    private func value(for name: String, in items: [URLQueryItem]) -> String? {
        items.first(where: { $0.name == name })?.value
    }

    private func parseCoordinatePair(from value: String) -> SdzSpotLocation? {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        let cleaned = trimmed.replacingOccurrences(of: " ", with: "")
        let parts = cleaned.split(separator: ",")
        guard parts.count >= 2,
              let lat = Double(parts[0]),
              let lng = Double(parts[1]) else {
            return nil
        }
        return SdzSpotLocation(lat: lat, lng: lng)
    }

    private func parseCoordinateFromPath(_ value: String) -> SdzSpotLocation? {
        let pattern = "@(-?\\d+\\.\\d+),(-?\\d+\\.\\d+)"
        guard let regex = try? NSRegularExpression(pattern: pattern, options: []) else {
            return nil
        }
        let range = NSRange(location: 0, length: value.utf16.count)
        guard let match = regex.firstMatch(in: value, options: [], range: range),
              match.numberOfRanges == 3,
              let latRange = Range(match.range(at: 1), in: value),
              let lngRange = Range(match.range(at: 2), in: value) else {
            return nil
        }
        let latString = String(value[latRange])
        let lngString = String(value[lngRange])
        guard let lat = Double(latString), let lng = Double(lngString) else {
            return nil
        }
        return SdzSpotLocation(lat: lat, lng: lng)
    }

    private func resolveRedirectUrl(urlString: String) async -> URL? {
        guard let url = URL(string: urlString) else {
            return nil
        }
        do {
            let (_, response) = try await URLSession.shared.data(from: url)
            return response.url
        } catch {
            return nil
        }
    }

    func toggleFavorite(_ spot: SdzSpot) async {
        guard let idToken = idToken else {
            favoritesErrorMessage = "ログインが必要です。"
            return
        }
        favoritesErrorMessage = nil
        let isAdding = !isFavorite(spot)
        if isAdding {
            favoriteSpots.insert(spot, at: 0)
        } else {
            favoriteSpots.removeAll { $0.spotId == spot.spotId }
        }
        saveFavorites()

        let apiClient = SdzApiClient(environment: environment, idToken: idToken)
        do {
            if isAdding {
                _ = try await apiClient.addToMyList(spotId: spot.spotId)
            } else {
                _ = try await apiClient.removeFromMyList(spotId: spot.spotId)
            }
        } catch {
            // Revert on failure to keep state consistent.
            if isAdding {
                favoriteSpots.removeAll { $0.spotId == spot.spotId }
            } else {
                favoriteSpots.insert(spot, at: 0)
            }
            favoritesErrorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
            saveFavorites()
        }
    }

    func isFavorite(_ spot: SdzSpot) -> Bool {
        favoriteSpots.contains(where: { $0.spotId == spot.spotId })
    }

    private func applySession(_ session: SdzAuthSession?) {
        if let session = session {
            idToken = session.idToken
            authUserId = session.userId
            authEmail = session.email
            authDisplayName = session.displayName
            authProviderIds = session.providerIds
            isAuthenticated = true
            Task {
                await refreshFavorites()
            }
        } else {
            idToken = nil
            authUserId = nil
            authEmail = nil
            authDisplayName = nil
            authProviderIds = []
            isAuthenticated = false
            favoriteSpots = []
            favoritesErrorMessage = nil
            saveFavorites()
        }
    }

    private func loadFavorites() {
        guard let data = UserDefaults.standard.data(forKey: Self.favoritesKey) else {
            return
        }
        if let decoded = try? JSONDecoder().decode([SdzSpot].self, from: data) {
            favoriteSpots = decoded
        }
    }

    private func saveFavorites() {
        guard let data = try? JSONEncoder().encode(favoriteSpots) else {
            return
        }
        UserDefaults.standard.set(data, forKey: Self.favoritesKey)
    }

    private static let favoritesKey = "sdz.favoriteSpots"

    func refreshFavorites() async {
        guard let idToken = idToken else {
            favoriteSpots = []
            favoritesErrorMessage = "ログインが必要です。"
            saveFavorites()
            return
        }
        isFavoritesLoading = true
        favoritesErrorMessage = nil
        let apiClient = SdzApiClient(environment: environment, idToken: idToken)
        do {
            let spots = try await apiClient.fetchMyList()
            favoriteSpots = spots
            saveFavorites()
        } catch {
            favoritesErrorMessage = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
        }
        isFavoritesLoading = false
    }

    func setProfileImageData(_ data: Data?) {
        profileImageData = data
        saveProfileImage()
    }

    private func loadProfileImage() {
        profileImageData = UserDefaults.standard.data(forKey: Self.profileImageKey)
    }

    private func saveProfileImage() {
        if let data = profileImageData {
            UserDefaults.standard.set(data, forKey: Self.profileImageKey)
        } else {
            UserDefaults.standard.removeObject(forKey: Self.profileImageKey)
        }
    }

    private static let profileImageKey = "sdz.profileImage"
}
