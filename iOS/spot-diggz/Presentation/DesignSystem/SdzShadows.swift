import SwiftUI

/// Design system shadow levels for SpotDiggz.
enum SdzShadowLevel {
    /// Small shadow - pins, small elements
    case sm
    /// Medium shadow - cards, search bars
    case md
    /// Large shadow - FABs, modals
    case lg
}

/// ViewModifier for applying consistent shadow styling.
struct SdzShadowModifier: ViewModifier {
    let level: SdzShadowLevel

    func body(content: Content) -> some View {
        switch level {
        case .sm:
            content.shadow(color: Color.black.opacity(0.08), radius: 2, x: 0, y: 1)
        case .md:
            content.shadow(color: Color.black.opacity(0.12), radius: 6, x: 0, y: 3)
        case .lg:
            content.shadow(color: Color.black.opacity(0.16), radius: 10, x: 0, y: 4)
        }
    }
}

extension View {
    /// Applies a design system shadow at the specified level.
    func sdzShadow(_ level: SdzShadowLevel) -> some View {
        modifier(SdzShadowModifier(level: level))
    }
}
