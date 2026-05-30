import SwiftUI

/// A design system search bar with magnifying glass icon.
struct SdzSearchBar: View {
    let placeholder: String
    @Binding var text: String
    var tintColor: Color = .sdzStreet

    var body: some View {
        HStack(spacing: SdzSpacing.sm + 2) {
            Image(systemName: "magnifyingglass")
                .foregroundColor(tintColor)
            TextField(placeholder, text: $text)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled(true)
        }
        .padding(.horizontal, SdzSpacing.md + 2)
        .padding(.vertical, SdzSpacing.md)
        .background(
            RoundedRectangle(cornerRadius: SdzRadius.md, style: .continuous)
                .fill(Color(.systemBackground))
        )
        .overlay(
            RoundedRectangle(cornerRadius: SdzRadius.md, style: .continuous)
                .stroke(tintColor.opacity(0.35), lineWidth: 1.5)
        )
        .sdzShadow(.md)
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

#if DEBUG
struct SdzSearchBar_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 16) {
            SdzSearchBar(placeholder: "スポットを検索", text: .constant(""))
            SdzSearchBar(placeholder: "場所を検索", text: .constant("渋谷"), tintColor: .sdzPark)
        }
        .padding()
        .previewLayout(.sizeThatFits)
    }
}
#endif
