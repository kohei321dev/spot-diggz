import SwiftUI

/// A card representing a single skate spot in the list.
struct SpotCardView: View {
    let spot: SdzSpot

    var body: some View {
        HStack(alignment: .top) {
            if let firstUrl = spot.images.first, let url = URL(string: firstUrl) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .empty:
                        ProgressView()
                            .frame(width: 80, height: 80)
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFill()
                            .frame(width: 80, height: 80)
                            .clipped()
                            .cornerRadius(SdzRadius.sm)
                    case .failure:
                        placeholderImage
                    @unknown default:
                        placeholderImage
                    }
                }
            } else {
                placeholderImage
            }

            VStack(alignment: .leading, spacing: SdzSpacing.xs) {
                Text(spot.name)
                    .font(SdzTypography.headline)
                    .foregroundColor(.sdzTextPrimary)
                if let description = spot.description, !description.isEmpty {
                    Text(description)
                        .font(SdzTypography.subheadline)
                        .foregroundColor(.sdzTextSecondary)
                        .lineLimit(2)
                }
                if !spot.tags.isEmpty {
                    Text(spot.tags.joined(separator: ", "))
                        .font(SdzTypography.caption1)
                        .foregroundColor(.sdzTextTertiary)
                }
            }
            Spacer()
            SdzStatusBadge(status: spot.approvalStatus, style: .compact)
        }
        .padding(.vertical, SdzSpacing.sm)
    }

    private var placeholderImage: some View {
        Rectangle()
            .fill(Color.sdzBgTertiary)
            .frame(width: 80, height: 80)
            .cornerRadius(SdzRadius.sm)
    }
}

#if DEBUG
struct SpotCardView_Previews: PreviewProvider {
    static var previews: some View {
        SpotCardView(spot: SdzSpot.sample(id: "sample", name: "サンプルスポット"))
            .previewLayout(.sizeThatFits)
    }
}
#endif
