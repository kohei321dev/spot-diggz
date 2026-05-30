import SwiftUI

/// Display style for the status badge.
enum SdzStatusBadgeStyle {
    /// Pill-shaped badge with background color
    case pill
    /// Compact text-only badge
    case compact
    /// Icon-only badge
    case icon
}

/// A badge indicating the approval status of a spot.
struct SdzStatusBadge: View {
    let status: SdzSpotApprovalStatus?
    var style: SdzStatusBadgeStyle = .pill

    var body: some View {
        switch style {
        case .pill:
            pillView
        case .compact:
            compactView
        case .icon:
            iconView
        }
    }

    private var pillView: some View {
        HStack(spacing: SdzSpacing.xs) {
            Image(systemName: iconName)
                .font(SdzTypography.caption2)
            Text(label)
                .font(SdzTypography.caption1Medium)
        }
        .foregroundColor(foregroundColor)
        .padding(.horizontal, SdzSpacing.sm)
        .padding(.vertical, SdzSpacing.xs)
        .background(backgroundColor)
        .clipShape(Capsule())
    }

    private var compactView: some View {
        Text(label)
            .font(SdzTypography.caption1)
            .foregroundColor(foregroundColor)
    }

    private var iconView: some View {
        Image(systemName: iconName)
            .font(SdzTypography.caption1)
            .foregroundColor(foregroundColor)
    }

    private var label: String {
        switch status {
        case .approved:
            return "承認済"
        case .pending:
            return "審査中"
        case .rejected:
            return "差戻し"
        case .none:
            return "未申請"
        }
    }

    private var iconName: String {
        switch status {
        case .approved:
            return "checkmark.seal.fill"
        case .pending:
            return "clock.fill"
        case .rejected:
            return "xmark.circle.fill"
        case .none:
            return "doc.text"
        }
    }

    private var foregroundColor: Color {
        switch status {
        case .approved:
            return .sdzSuccess
        case .pending:
            return .sdzWarning
        case .rejected:
            return .sdzError
        case .none:
            return .sdzTextTertiary
        }
    }

    private var backgroundColor: Color {
        foregroundColor.opacity(0.12)
    }
}

#if DEBUG
struct SdzStatusBadge_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 16) {
            SdzStatusBadge(status: .approved)
            SdzStatusBadge(status: .pending)
            SdzStatusBadge(status: .rejected)
            SdzStatusBadge(status: nil)
            SdzStatusBadge(status: .approved, style: .compact)
            SdzStatusBadge(status: .pending, style: .icon)
        }
        .padding()
        .previewLayout(.sizeThatFits)
    }
}
#endif
