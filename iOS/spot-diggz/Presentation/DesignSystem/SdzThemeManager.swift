import SwiftUI

/// User preference for color scheme.
enum SdzColorSchemePreference: String, CaseIterable, Identifiable {
    case system
    case light
    case dark

    var id: String { rawValue }

    var label: String {
        switch self {
        case .system:
            return "システム"
        case .light:
            return "ライト"
        case .dark:
            return "ダーク"
        }
    }

    var iconName: String {
        switch self {
        case .system:
            return "circle.lefthalf.filled"
        case .light:
            return "sun.max.fill"
        case .dark:
            return "moon.fill"
        }
    }
}

/// Manages color scheme preference with persistence via UserDefaults.
@Observable
final class SdzThemeManager {
    var preference: SdzColorSchemePreference {
        didSet {
            UserDefaults.standard.set(preference.rawValue, forKey: "sdz_color_scheme")
        }
    }

    init() {
        let raw = UserDefaults.standard.string(forKey: "sdz_color_scheme")
            ?? SdzColorSchemePreference.system.rawValue
        self.preference = SdzColorSchemePreference(rawValue: raw) ?? .system
    }

    /// Resolved color scheme for `.preferredColorScheme()`.
    /// Returns `nil` for system (follows device setting).
    var resolvedColorScheme: ColorScheme? {
        switch preference {
        case .system:
            return nil
        case .light:
            return .light
        case .dark:
            return .dark
        }
    }
}
