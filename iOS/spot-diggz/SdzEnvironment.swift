import Foundation

/// Represents different API environments for SpotDiggz.
/// Use this enum to switch between local, development, and production endpoints.
enum SdzEnvironment: String {
    case local
    case dev
    case prod

    /// Base URL for the selected environment.
    var baseURL: URL {
        switch self {
        case .local:
            return URL(string: "http://localhost:8080")!
        case .dev:
            return URL(string: "https://sdz-dev-api-1053202159855.asia-northeast1.run.app")!
        case .prod:
            return URL(string: "https://sdz-api.example.com")!
        }
    }
}
