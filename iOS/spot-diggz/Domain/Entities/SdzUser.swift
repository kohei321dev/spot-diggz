import Foundation

/// Represents a user of the SpotDiggz platform.
struct SdzUser: Codable, Identifiable {
    let userId: String
    let displayName: String
    let email: String?

    var id: String { userId }
}