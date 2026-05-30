import Foundation

/// Input used when updating an existing spot.
struct SdzUpdateSpotInput: Codable {
    let name: String
    let description: String?
    let location: SdzSpotLocation?
    let tags: [String]?
    let images: [String]?
    let approvalStatus: SdzSpotApprovalStatus?
    let parkAttributes: SdzSpotParkAttributes?
    let streetAttributes: SdzStreetAttributes?
}
