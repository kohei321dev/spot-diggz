import SwiftUI
import WebKit

struct SdzUrlPickerSheetView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var currentUrl: String
    private let onSelect: (String) -> Void

    init(initialUrl: String, onSelect: @escaping (String) -> Void) {
        _currentUrl = State(initialValue: initialUrl)
        self.onSelect = onSelect
    }

    var body: some View {
        NavigationStack {
            SdzUrlWebView(urlString: $currentUrl)
                .navigationTitle("公式サイトを探す")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("閉じる") {
                            dismiss()
                        }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("このURLを追加") {
                            let trimmed = currentUrl.trimmingCharacters(in: .whitespacesAndNewlines)
                            guard !trimmed.isEmpty else {
                                return
                            }
                            onSelect(trimmed)
                            dismiss()
                        }
                    }
                }
        }
    }
}

struct SdzUrlWebView: UIViewRepresentable {
    @Binding var urlString: String

    func makeUIView(context: Context) -> WKWebView {
        let webView = WKWebView()
        webView.navigationDelegate = context.coordinator
        webView.allowsBackForwardNavigationGestures = true
        let initial = urlString.isEmpty ? defaultUrl() : urlString
        if let url = URL(string: initial) {
            webView.load(URLRequest(url: url))
        }
        return webView
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    final class Coordinator: NSObject, WKNavigationDelegate {
        private let parent: SdzUrlWebView

        init(parent: SdzUrlWebView) {
            self.parent = parent
        }

        func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
            if let url = webView.url?.absoluteString {
                parent.urlString = url
            }
        }
    }

    private func defaultUrl() -> String {
        "https://www.google.com"
    }
}
