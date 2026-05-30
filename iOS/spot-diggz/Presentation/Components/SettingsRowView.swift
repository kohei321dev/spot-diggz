import SwiftUI

/// A reusable settings row with icon, title, optional value and optional chevron.
/// Delegates rendering to the design system `SdzSettingsRow`.
struct SettingsRowView: View {
    let iconName: String
    let title: String
    let value: String?
    let showsChevron: Bool
    let isDestructive: Bool

    init(
        iconName: String,
        title: String,
        value: String? = nil,
        showsChevron: Bool = true,
        isDestructive: Bool = false
    ) {
        self.iconName = iconName
        self.title = title
        self.value = value
        self.showsChevron = showsChevron
        self.isDestructive = isDestructive
    }

    var body: some View {
        SdzSettingsRow(
            iconName: iconName,
            title: title,
            value: value,
            showsChevron: showsChevron,
            isDestructive: isDestructive
        )
    }
}

#if DEBUG
struct SettingsRowView_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 0) {
            SettingsRowView(iconName: "person.circle", title: "アカウント情報")
            SdzDividerView()
            SettingsRowView(iconName: "bell", title: "通知", value: "オン")
            SdzDividerView()
            SettingsRowView(iconName: "rectangle.portrait.and.arrow.right", title: "ログアウト", showsChevron: false, isDestructive: true)
        }
        .background { Color.sdzSurface }
        .previewLayout(.sizeThatFits)
    }
}
#endif
