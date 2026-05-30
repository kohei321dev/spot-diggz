import SwiftUI

/// Design system spacing scale for SpotDiggz.
///
/// Based on an 8pt grid with additional smaller increments.
enum SdzSpacing {
    /// 2pt - Extremely tight spacing
    static let xxs: CGFloat = 2

    /// 4pt - Tight spacing between related elements
    static let xs: CGFloat = 4

    /// 8pt - Standard small spacing
    static let sm: CGFloat = 8

    /// 12pt - Medium spacing
    static let md: CGFloat = 12

    /// 16pt - Standard component spacing
    static let lg: CGFloat = 16

    /// 24pt - Large spacing between sections
    static let xl: CGFloat = 24

    /// 32pt - Extra large margins
    static let xxl: CGFloat = 32
}
