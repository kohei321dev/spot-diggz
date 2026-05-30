import Foundation

/// Input used when creating a new spot.
struct SdzCreateSpotInput: Codable {
    let name: String
    let description: String?
    let location: SdzSpotLocation?
    let tags: [String]?
    let images: [String]?
    let parkAttributes: SdzSpotParkAttributes?
    let streetAttributes: SdzStreetAttributes?
}
