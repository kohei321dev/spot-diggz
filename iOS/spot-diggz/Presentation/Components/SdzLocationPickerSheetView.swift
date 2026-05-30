import SwiftUI

/// Full-screen location picker wrapper for better pin placement.
struct SdzLocationPickerSheetView: View {
    @Binding var selectedLocation: SdzSpotLocation?
    var onRequestCurrentLocation: (() -> Void)? = nil
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        Group {
            if #available(iOS 16.0, *) {
                NavigationStack {
                    content
                }
            } else {
                NavigationView {
                    content
                }
            }
        }
    }

    private var content: some View {
        SdzLocationPickerView(
            selectedLocation: $selectedLocation,
            mode: .fullScreen,
            onRequestCurrentLocation: onRequestCurrentLocation,
            onConfirmSelection: { _ in
                dismiss()
            }
        )
            .navigationTitle("位置を選択")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("キャンセル") {
                        dismiss()
                    }
                }
            }
            .ignoresSafeArea(edges: .bottom)
    }
}

#if DEBUG
struct SdzLocationPickerSheetView_Previews: PreviewProvider {
    static var previews: some View {
        SdzLocationPickerSheetView(selectedLocation: .constant(nil))
            .environmentObject(SdzAppState())
    }
}
#endif
