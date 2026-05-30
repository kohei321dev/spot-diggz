import SwiftUI

/// Design system animation constants for SpotDiggz.
enum SdzAnimation {
    /// Quick interactions (button press, toggle)
    static let fast = Animation.easeInOut(duration: 0.15)

    /// Standard transitions (expand/collapse, tab switch)
    static let standard = Animation.easeInOut(duration: 0.24)

    /// Smooth entrance/exit (sheet, modal)
    static let smooth = Animation.easeInOut(duration: 0.35)

    /// Spring animation for interactive elements (pin focus, card highlight)
    static let spring = Animation.spring(response: 0.25, dampingFraction: 0.8)
}
