import Foundation
import SwiftUI

/// Location information for a spot.
struct SdzSpotLocation: Codable, Identifiable, Equatable {
    let lat: Double
    let lng: Double

    // Use a stable identifier derived from coordinates.
    var id: String { "\(lat),\(lng)" }
}

struct SdzSpotTimeRange: Codable, Equatable {
    let startMinutes: Int
    let endMinutes: Int
}

enum SdzSpotBusinessScheduleType: String, Codable, CaseIterable, Identifiable {
    case regular
    case weekdayOnly
    case weekendOnly
    case irregular
    case schoolOnly
    case manual

    var id: String { rawValue }

    var label: String {
        switch self {
        case .regular:
            return "通常"
        case .weekdayOnly:
            return "平日のみ"
        case .weekendOnly:
            return "週末のみ"
        case .irregular:
            return "不定休"
        case .schoolOnly:
            return "スクールのみ"
        case .manual:
            return "手動入力"
        }
    }
}

struct SdzSpotBusinessHours: Codable, Equatable {
    let scheduleType: SdzSpotBusinessScheduleType?
    let is24Hours: Bool
    let sameAsWeekday: Bool
    let weekday: SdzSpotTimeRange?
    let weekend: SdzSpotTimeRange?
    let note: String?
}

struct SdzSpotParkAttributes: Codable, Equatable {
    let officialUrl: String?
    let businessHours: SdzSpotBusinessHours?
    let accessInfo: String?
    let phoneNumber: String?
}

struct SdzStreetSurfaceCondition: Codable, Equatable {
    let roughness: String?
    let crack: String?
}

struct SdzStreetSection: Codable, Equatable {
    let type: String
    let count: Int?
    let heightCm: Int?
    let widthCm: Int?
    let notes: String?
}

struct SdzStreetAttributes: Codable, Equatable {
    let surfaceMaterial: String?
    let surfaceCondition: SdzStreetSurfaceCondition?
    let sections: [SdzStreetSection]?
    let difficulty: String?
    let notes: String?
}

/// Approval status for a spot.
enum SdzSpotApprovalStatus: String, Codable {
    case pending
    case approved
    case rejected
}

/// Represents a skate spot posted by a user.
struct SdzSpot: Codable, Identifiable {
    let spotId: String
    let name: String
    let description: String?
    let location: SdzSpotLocation?
    let tags: [String]
    let images: [String]
    let approvalStatus: SdzSpotApprovalStatus?
    let parkAttributes: SdzSpotParkAttributes?
    let streetAttributes: SdzStreetAttributes?
    let userId: String
    let createdAt: Date
    let updatedAt: Date

    var id: String { spotId }
}

extension SdzSpot {
    /// Whether this spot is a park (based on parkAttributes or tags).
    var sdzIsPark: Bool {
        if parkAttributes != nil {
            return true
        }
        return tags.contains("パーク")
    }

    /// The design system color for this spot's pin.
    var sdzPinColor: Color {
        sdzIsPark ? .sdzPark : .sdzStreet
    }

    /// The UIColor variant for MapKit annotation views.
    var sdzPinUIColor: UIColor {
        sdzIsPark ? .sdzPark : .sdzStreet
    }

    /// A localized label for the spot type.
    var sdzTypeLabel: String {
        sdzIsPark ? "スケートパーク" : "ストリート"
    }

    /// Creates a sample spot for preview and development.
    static func sample(id: String, name: String) -> SdzSpot {
        return SdzSpot(
            spotId: id,
            name: name,
            description: "説明が入ります。このスポットは楽しい場所です。",
            location: SdzSpotLocation(lat: 34.67, lng: 135.5),
            tags: ["パーク", "ストリート"],
            images: [],
            approvalStatus: nil,
            parkAttributes: nil,
            streetAttributes: nil,
            userId: "user",
            createdAt: Date(),
            updatedAt: Date()
        )
    }
}
