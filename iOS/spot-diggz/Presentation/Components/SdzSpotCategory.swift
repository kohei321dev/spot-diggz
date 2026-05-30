import Foundation

enum SdzSpotCategory: String, CaseIterable, Identifiable {
    case street
    case park

    var id: String { rawValue }

    var label: String {
        switch self {
        case .street:
            return "ストリート"
        case .park:
            return "スケートパーク"
        }
    }

    var defaultTag: String {
        switch self {
        case .street:
            return "ストリート"
        case .park:
            return "パーク"
        }
    }
}
