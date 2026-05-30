import SwiftUI

/// Button style variants for the design system.
enum SdzButtonVariant {
    case primary
    case secondary
    case ghost
    case iconOnly
    case destructive
}

/// A design system button with consistent styling.
struct SdzButton: View {
    let title: String
    var icon: String?
    var variant: SdzButtonVariant = .primary
    var isLoading: Bool = false
    var isDisabled: Bool = false
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: SdzSpacing.sm) {
                if isLoading {
                    ProgressView()
                        .tint(foregroundColor)
                }
                if let icon {
                    Image(systemName: icon)
                        .font(SdzTypography.subheadlineMedium)
                }
                if variant != .iconOnly {
                    Text(title)
                        .font(SdzTypography.subheadlineMedium)
                }
            }
            .foregroundColor(foregroundColor)
            .frame(maxWidth: variant == .iconOnly ? nil : .infinity)
            .padding(.horizontal, variant == .iconOnly ? SdzSpacing.md : SdzSpacing.lg)
            .padding(.vertical, SdzSpacing.md)
            .background(backgroundColor)
            .clipShape(RoundedRectangle(cornerRadius: SdzRadius.sm, style: .continuous))
            .overlay(borderOverlay)
        }
        .disabled(isDisabled || isLoading)
        .opacity(isDisabled ? 0.5 : 1)
    }

    private var foregroundColor: Color {
        switch variant {
        case .primary:
            return .white
        case .secondary:
            return .sdzTextPrimary
        case .ghost:
            return .sdzStreet
        case .iconOnly:
            return .sdzTextPrimary
        case .destructive:
            return .white
        }
    }

    private var backgroundColor: Color {
        switch variant {
        case .primary:
            return .sdzStreet
        case .secondary:
            return .sdzBgSecondary
        case .ghost, .iconOnly:
            return .clear
        case .destructive:
            return .sdzError
        }
    }

    @ViewBuilder
    private var borderOverlay: some View {
        if variant == .secondary {
            RoundedRectangle(cornerRadius: SdzRadius.sm, style: .continuous)
                .stroke(Color.sdzBorder, lineWidth: 1)
        }
    }
}

#if DEBUG
struct SdzButton_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 16) {
            SdzButton(title: "Primary", action: {})
            SdzButton(title: "Secondary", variant: .secondary, action: {})
            SdzButton(title: "Ghost", variant: .ghost, action: {})
            SdzButton(title: "Destructive", variant: .destructive, action: {})
            SdzButton(title: "Loading", isLoading: true, action: {})
            SdzButton(title: "", icon: "plus", variant: .iconOnly, action: {})
        }
        .padding()
        .previewLayout(.sizeThatFits)
    }
}
#endif
