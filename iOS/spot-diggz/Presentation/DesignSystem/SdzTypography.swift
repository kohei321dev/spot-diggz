import SwiftUI

/// Design system typography scale for SpotDiggz.
///
/// Based on SF Pro with a defined hierarchy of sizes and weights.
enum SdzTypography {
    /// 34pt bold - App name, hero text
    static let display: Font = .system(size: 34, weight: .bold)

    /// 28pt bold - Large headings
    static let title1: Font = .system(size: 28, weight: .bold)

    /// 22pt bold - Section headings
    static let title2: Font = .system(size: 22, weight: .bold)

    /// 20pt semibold - Subsection headings
    static let title3: Font = .system(size: 20, weight: .semibold)

    /// 17pt semibold - Card titles, emphasized body
    static let headline: Font = .system(size: 17, weight: .semibold)

    /// 17pt regular - Primary body text
    static let body: Font = .system(size: 17, weight: .regular)

    /// 17pt medium - Emphasized body text
    static let bodyMedium: Font = .system(size: 17, weight: .medium)

    /// 15pt regular - Secondary text
    static let subheadline: Font = .system(size: 15, weight: .regular)

    /// 15pt medium - Emphasized secondary text (replaces size: 29/2)
    static let subheadlineMedium: Font = .system(size: 15, weight: .medium)

    /// 13pt regular - Captions, metadata
    static let caption1: Font = .system(size: 13, weight: .regular)

    /// 13pt medium - Emphasized captions
    static let caption1Medium: Font = .system(size: 13, weight: .medium)

    /// 11pt regular - Smallest text, fine print
    static let caption2: Font = .system(size: 11, weight: .regular)
}
