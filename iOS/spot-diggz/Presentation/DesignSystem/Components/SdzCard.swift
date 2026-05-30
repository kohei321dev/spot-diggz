import SwiftUI

/// Card surface style variants.
enum SdzCardStyle {
    /// Standard surface card
    case surface
    /// Elevated card with shadow
    case elevated
    /// Material-backed card (blur effect)
    case material
}

/// A design system card container.
struct SdzCard<Content: View>: View {
    let style: SdzCardStyle
    let content: Content

    init(style: SdzCardStyle = .surface, @ViewBuilder content: () -> Content) {
        self.style = style
        self.content = content()
    }

    var body: some View {
        Group {
            switch style {
            case .surface:
                content
                    .background(Color.sdzSurface)
                    .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
                    .overlay(
                        RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous)
                            .stroke(Color.sdzBorder, lineWidth: 1)
                    )
            case .elevated:
                content
                    .background(Color.sdzSurfaceElevated)
                    .clipShape(RoundedRectangle(cornerRadius: SdzRadius.xl, style: .continuous))
                    .sdzShadow(.md)
            case .material:
                content
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: SdzRadius.md, style: .continuous))
            }
        }
    }
}

#if DEBUG
struct SdzCard_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 16) {
            SdzCard(style: .surface) {
                Text("Surface Card")
                    .padding()
            }
            SdzCard(style: .elevated) {
                Text("Elevated Card")
                    .padding()
            }
            SdzCard(style: .material) {
                Text("Material Card")
                    .padding()
            }
        }
        .padding()
        .background(Color.sdzBgSecondary)
        .previewLayout(.sizeThatFits)
    }
}
#endif
