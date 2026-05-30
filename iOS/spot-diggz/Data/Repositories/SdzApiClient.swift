import Foundation

struct SdzErrorResponse: Codable {
    let code: Int
    let errorCode: String
    let message: String
}

enum SdzApiError: LocalizedError {
    case invalidUrl
    case invalidResponse
    case api(statusCode: Int, error: SdzErrorResponse)
    case statusCode(Int)
    case decoding(Error)
    case network(Error)
    case authRequired

    var errorDescription: String? {
        switch self {
        case .invalidUrl:
            return "APIのURLが不正です。"
        case .invalidResponse:
            return "APIのレスポンスが不正です。"
        case let .api(statusCode, error):
            return "\(error.message) (HTTP \(statusCode))"
        case let .statusCode(code):
            return "サーバーエラーが発生しました (HTTP \(code))"
        case let .decoding(error):
            return "レスポンスの解析に失敗しました: \(error.localizedDescription)"
        case let .network(error):
            return "通信に失敗しました: \(error.localizedDescription)"
        case .authRequired:
            return "ログインが必要です。"
        }
    }
}

/// API client for communicating with the SpotDiggz backend.
final class SdzApiClient {
    private let environment: SdzEnvironment
    private let urlSession: URLSession
    private let idToken: String?

    init(environment: SdzEnvironment, idToken: String? = nil, urlSession: URLSession = .shared) {
        self.environment = environment
        self.idToken = idToken
        self.urlSession = urlSession
    }

    /// Fetches spots with optional search query.
    func fetchSpots(
        query: SdzSpotSearchQuery? = nil,
        includeAuth: Bool = false
    ) async throws -> [SdzSpot] {
        try await request(
            path: "/sdz/spots",
            queryItems: query?.queryItems,
            includeAuthIfAvailable: includeAuth
        )
    }

    /// Fetches a specific spot by ID.
    func fetchSpotDetail(id: String, includeAuth: Bool = false) async throws -> SdzSpot {
        try await request(path: "/sdz/spots/\(id)", includeAuthIfAvailable: includeAuth)
    }

    /// Creates a new spot.
    func createSpot(_ input: SdzCreateSpotInput) async throws -> SdzSpot {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        let body = try JSONEncoder().encode(input)
        return try await request(
            path: "/sdz/spots",
            method: "POST",
            body: body,
            requiresAuth: true,
            requiresMobileClient: true
        )
    }

    /// Updates an existing spot.
    func updateSpot(id: String, input: SdzUpdateSpotInput) async throws -> SdzSpot {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        let body = try JSONEncoder().encode(input)
        return try await request(
            path: "/sdz/spots/\(id)",
            method: "PATCH",
            body: body,
            requiresAuth: true,
            requiresMobileClient: true
        )
    }

    /// Fetches current user.
    func fetchCurrentUser() async throws -> SdzUser {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        return try await request(path: "/sdz/users/me", requiresAuth: true)
    }

    /// Requests a signed upload URL for spot images.
    func requestUploadUrl(contentType: String) async throws -> SdzUploadUrlResponse {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        let payload = SdzUploadUrlRequest(contentType: contentType)
        let body = try JSONEncoder().encode(payload)
        return try await request(
            path: "/sdz/spots/upload-url",
            method: "POST",
            body: body,
            requiresAuth: true,
            requiresMobileClient: true
        )
    }

    /// Fetches the current user's mylist spots.
    func fetchMyList() async throws -> [SdzSpot] {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        do {
            return try await request(path: "/sdz/mylist", requiresAuth: true)
        } catch let error as SdzApiError {
            switch error {
            case .statusCode(404), .api(statusCode: 404, error: _):
                return []
            default:
                throw error
            }
        }
    }

    /// Adds a spot to the current user's mylist.
    func addToMyList(spotId: String) async throws -> SdzMyListActionResponse {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        let payload = SdzMyListActionRequest(spotId: spotId)
        let body = try JSONEncoder().encode(payload)
        return try await request(
            path: "/sdz/mylist",
            method: "POST",
            body: body,
            requiresAuth: true
        )
    }

    /// Removes a spot from the current user's mylist.
    func removeFromMyList(spotId: String) async throws -> SdzMyListActionResponse {
        guard idToken != nil else {
            throw SdzApiError.authRequired
        }
        return try await request(
            path: "/sdz/mylist/\(spotId)",
            method: "DELETE",
            requiresAuth: true
        )
    }

    /// Uploads binary image data to the signed URL.
    func uploadImage(data: Data, contentType: String, uploadUrl: String) async throws {
        guard let url = URL(string: uploadUrl) else {
            throw SdzApiError.invalidUrl
        }
        var request = URLRequest(url: url)
        request.httpMethod = "PUT"
        request.setValue(contentType, forHTTPHeaderField: "Content-Type")

        do {
            let (_, response) = try await urlSession.upload(for: request, from: data)
            guard let http = response as? HTTPURLResponse else {
                throw SdzApiError.invalidResponse
            }
            guard (200...299).contains(http.statusCode) else {
                throw SdzApiError.statusCode(http.statusCode)
            }
        } catch let error as SdzApiError {
            throw error
        } catch {
            throw SdzApiError.network(error)
        }
    }

    private func request<T: Decodable>(
        path: String,
        queryItems: [URLQueryItem]? = nil,
        method: String = "GET",
        body: Data? = nil,
        requiresAuth: Bool = false,
        includeAuthIfAvailable: Bool = false,
        requiresMobileClient: Bool = false
    ) async throws -> T {
        let request = try buildRequest(
            path: path,
            queryItems: queryItems,
            method: method,
            body: body,
            requiresAuth: requiresAuth,
            includeAuthIfAvailable: includeAuthIfAvailable,
            requiresMobileClient: requiresMobileClient
        )

        do {
            let (data, response) = try await urlSession.data(for: request)
            guard let http = response as? HTTPURLResponse else {
                throw SdzApiError.invalidResponse
            }

            if (200...299).contains(http.statusCode) {
                do {
                    return try decoder().decode(T.self, from: data)
                } catch {
                    throw SdzApiError.decoding(error)
                }
            }

            if let apiError = try? decoder().decode(SdzErrorResponse.self, from: data) {
                throw SdzApiError.api(statusCode: http.statusCode, error: apiError)
            }

            throw SdzApiError.statusCode(http.statusCode)
        } catch let error as SdzApiError {
            throw error
        } catch {
            throw SdzApiError.network(error)
        }
    }

    private func buildRequest(
        path: String,
        queryItems: [URLQueryItem]?,
        method: String,
        body: Data?,
        requiresAuth: Bool,
        includeAuthIfAvailable: Bool,
        requiresMobileClient: Bool
    ) throws -> URLRequest {
        let url = buildUrl(path: path, queryItems: queryItems)
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        if let body = body {
            request.httpBody = body
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }

        if requiresAuth {
            guard let token = idToken else {
                throw SdzApiError.authRequired
            }
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        } else if includeAuthIfAvailable, let token = idToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if requiresMobileClient {
            request.setValue("ios", forHTTPHeaderField: "X-SDZ-Client")
        }

        return request
    }

    private func buildUrl(path: String, queryItems: [URLQueryItem]?) -> URL {
        var normalized = path
        if normalized.hasPrefix("/") {
            normalized.removeFirst()
        }
        let baseUrl = environment.baseURL.appendingPathComponent(normalized)
        guard var components = URLComponents(url: baseUrl, resolvingAgainstBaseURL: false) else {
            return baseUrl
        }
        if let queryItems, !queryItems.isEmpty {
            components.queryItems = queryItems
        }
        return components.url ?? baseUrl
    }

    private func decoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let value = try container.decode(String.self)
            if let date = SdzApiClient.iso8601WithFractional.date(from: value)
                ?? SdzApiClient.iso8601Plain.date(from: value) {
                return date
            }
            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Invalid date format: \(value)"
            )
        }
        return decoder
    }

    private static let iso8601WithFractional: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    private static let iso8601Plain: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()
}

struct SdzUploadUrlRequest: Codable {
    let contentType: String
}

struct SdzUploadUrlResponse: Codable {
    let uploadUrl: String
    let objectUrl: String
    let objectName: String
    let expiresAt: Date
}

struct SdzMyListActionRequest: Codable {
    let spotId: String
}

struct SdzMyListActionResponse: Codable {
    let spotId: String
    let status: String
}
