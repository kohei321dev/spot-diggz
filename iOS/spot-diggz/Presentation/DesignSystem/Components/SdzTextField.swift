import SwiftUI

/// A design system text field with consistent styling.
struct SdzTextField: View {
    let placeholder: String
    @Binding var text: String
    var axis: Axis = .horizontal

    var body: some View {
        TextField(placeholder, text: $text, axis: axis)
            .font(SdzTypography.body)
            .padding(.horizontal, SdzSpacing.md)
            .padding(.vertical, SdzSpacing.md)
            .background(Color.sdzSurface)
            .clipShape(RoundedRectangle(cornerRadius: SdzRadius.sm, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: SdzRadius.sm, style: .continuous)
                    .stroke(Color.sdzBorder, lineWidth: 1)
            )
    }
}

#if DEBUG
struct SdzTextField_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 16) {
            SdzTextField(placeholder: "スポット名", text: .constant(""))
            SdzTextField(placeholder: "説明", text: .constant("テスト"), axis: .vertical)
        }
        .padding()
        .background(Color.sdzBgSecondary)
        .previewLayout(.sizeThatFits)
    }
}
#endif
