import SwiftUI

struct SdzSpotMiniCardView: View {
    let spot: SdzSpot

    var body: some View {
        HStack(spacing: SdzSpacing.md) {
            VStack(alignment: .leading, spacing: SdzSpacing.xs) {
                Text(spot.name)
                    .font(SdzTypography.headline)
                    .foregroundColor(.sdzTextPrimary)
                    .lineLimit(1)
                Text(spot.sdzTypeLabel)
                    .font(SdzTypography.caption1)
                    .foregroundColor(.sdzTextSecondary)
            }
            Spacer()
            Image(systemName: "chevron.up")
                .font(SdzTypography.caption1)
                .foregroundColor(.sdzTextTertiary)
        }
        .padding(.horizontal, SdzSpacing.md + 2)
        .padding(.vertical, SdzSpacing.sm + 2)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.lg, style: .continuous))
    }
}
