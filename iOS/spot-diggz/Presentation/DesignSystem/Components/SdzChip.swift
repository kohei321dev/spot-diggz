import SwiftUI

/// A filter chip for toggling selections (e.g., spot type, tags).
struct SdzChip: View {
    let title: String
    let isSelected: Bool
    var systemImage: String?
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: SdzSpacing.xs + 2) {
                if let systemImage {
                    Image(systemName: systemImage)
                        .font(SdzTypography.caption1)
                }
                Text(title)
                    .font(SdzTypography.caption1Medium)
            }
            .foregroundColor(isSelected ? .white : .sdzTextPrimary)
            .padding(.horizontal, SdzSpacing.md)
            .padding(.vertical, SdzSpacing.xs + 2)
            .background(isSelected ? Color.sdzStreet : Color.sdzSurface)
            .clipShape(Capsule())
            .overlay(
                Capsule()
                    .stroke(isSelected ? Color.clear : Color.sdzBorder, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }
}

#if DEBUG
struct SdzChip_Previews: PreviewProvider {
    static var previews: some View {
        HStack(spacing: 8) {
            SdzChip(title: "すべて", isSelected: true, action: {})
            SdzChip(title: "パーク", isSelected: false, action: {})
            SdzChip(title: "タグ", isSelected: true, systemImage: "xmark.circle.fill", action: {})
        }
        .padding()
        .previewLayout(.sizeThatFits)
    }
}
#endif
