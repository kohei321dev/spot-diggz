import SwiftUI

/// A settings row component with icon, title, optional value and chevron.
struct SdzSettingsRow: View {
    let iconName: String
    let title: String
    var value: String?
    var showsChevron: Bool = true
    var isDestructive: Bool = false

    var body: some View {
        HStack(spacing: SdzSpacing.md + 2) {
            Image(systemName: iconName)
                .font(.system(size: 21, weight: .medium))
                .foregroundColor(isDestructive ? .sdzError : .sdzTextPrimary)
                .frame(width: 30)

            Text(title)
                .font(SdzTypography.subheadlineMedium)
                .foregroundColor(isDestructive ? .sdzError : .sdzTextPrimary)

            Spacer(minLength: 10)

            if let value {
                Text(value)
                    .font(SdzTypography.subheadlineMedium)
                    .foregroundColor(.sdzTextSecondary)
                    .lineLimit(1)
            }

            if showsChevron {
                Image(systemName: "chevron.right")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundColor(.sdzTextTertiary)
            }
        }
        .padding(.horizontal, SdzSpacing.lg + 2)
        .padding(.vertical, SdzSpacing.lg)
        .contentShape(Rectangle())
    }
}

#if DEBUG
struct SdzSettingsRow_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 0) {
            SdzSettingsRow(iconName: "person.circle", title: "アカウント情報")
            SdzDividerView()
            SdzSettingsRow(iconName: "bell", title: "通知", value: "オン")
            SdzDividerView()
            SdzSettingsRow(iconName: "rectangle.portrait.and.arrow.right", title: "ログアウト", showsChevron: false, isDestructive: true)
        }
        .background(Color.sdzSurface)
        .previewLayout(.sizeThatFits)
    }
}
#endif
