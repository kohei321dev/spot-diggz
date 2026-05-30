import SwiftUI

struct SdzStreetSectionInput: Identifiable, Equatable {
    let id: UUID
    var type: String
    var count: String
    var heightCm: String
    var widthCm: String
    var notes: String

    init(
        id: UUID = UUID(),
        type: String = "",
        count: String = "",
        heightCm: String = "",
        widthCm: String = "",
        notes: String = ""
    ) {
        self.id = id
        self.type = type
        self.count = count
        self.heightCm = heightCm
        self.widthCm = widthCm
        self.notes = notes
    }
}

struct SdzStreetSectionEditor: View {
    @Binding var section: SdzStreetSectionInput
    let onRemove: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                TextField("セクションタイプ (例: レール)", text: $section.type)
                Button(action: onRemove) {
                    Image(systemName: "trash")
                        .foregroundColor(.red)
                }
            }
            HStack {
                TextField("数", text: $section.count)
                    .keyboardType(.numberPad)
                TextField("高さ(cm)", text: $section.heightCm)
                    .keyboardType(.numberPad)
                TextField("幅(cm)", text: $section.widthCm)
                    .keyboardType(.numberPad)
            }
            TextField("メモ", text: $section.notes)
        }
        .padding(10)
        .background(Color(.secondarySystemBackground))
        .cornerRadius(10)
    }
}
