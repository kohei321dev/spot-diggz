import SwiftUI

/// A horizontally scrollable card for showcasing a spot on the map screen.
struct SpotCarouselCardView: View {
    let spot: SdzSpot
    let isFavorite: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: SdzSpacing.sm) {
            ZStack(alignment: .topTrailing) {
                coverImage
                if isFavorite {
                    Image(systemName: "heart.fill")
                        .foregroundColor(.pink)
                        .padding(SdzSpacing.sm)
                        .background(.ultraThinMaterial)
                        .clipShape(Circle())
                        .padding(SdzSpacing.sm)
                }
            }
            VStack(alignment: .leading, spacing: SdzSpacing.xs) {
                Text(spot.name)
                    .font(SdzTypography.headline)
                    .foregroundColor(.sdzTextPrimary)
                    .lineLimit(1)
                if let description = spot.description, !description.isEmpty {
                    Text(description)
                        .font(SdzTypography.caption1)
                        .foregroundColor(.sdzTextSecondary)
                        .lineLimit(2)
                }
                if !spot.tags.isEmpty {
                    Text(spot.tags.joined(separator: " / "))
                        .font(SdzTypography.caption2)
                        .foregroundColor(.sdzTextTertiary)
                }
            }
            .padding(.horizontal, SdzSpacing.md)
            .padding(.bottom, SdzSpacing.md)
        }
        .frame(width: 280)
        .background(.ultraThinMaterial)
        .clipShape(RoundedRectangle(cornerRadius: SdzRadius.lg, style: .continuous))
        .sdzShadow(.md)
    }

    private var coverImage: some View {
        Group {
            if let firstUrl = spot.images.first, let url = URL(string: firstUrl) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .empty:
                        ProgressView()
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFill()
                    case .failure:
                        placeholderImage
                    @unknown default:
                        placeholderImage
                    }
                }
            } else {
                placeholderImage
            }
        }
        .frame(height: 170)
        .clipped()
    }

    private var placeholderImage: some View {
        Rectangle()
            .fill(Color.sdzBgTertiary)
            .overlay(
                Image(systemName: "photo")
                    .foregroundColor(.sdzTextTertiary)
            )
    }
}

#if DEBUG
struct SpotCarouselCardView_Previews: PreviewProvider {
    static var previews: some View {
        SpotCarouselCardView(spot: SdzSpot.sample(id: "sample", name: "サンプルスポット"), isFavorite: true)
            .previewLayout(.sizeThatFits)
            .padding()
    }
}
#endif
