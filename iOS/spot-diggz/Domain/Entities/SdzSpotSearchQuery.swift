import Foundation

struct SdzSpotSearchQuery {
    let text: String?
    let spotType: String?
    let tags: [String]

    init(text: String? = nil, spotType: String? = nil, tags: [String] = []) {
        self.text = text?.trimmingCharacters(in: .whitespacesAndNewlines)
        self.spotType = spotType?.trimmingCharacters(in: .whitespacesAndNewlines)
        self.tags = tags
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }

    var queryItems: [URLQueryItem] {
        var items: [URLQueryItem] = []
        if let text = text, !text.isEmpty {
            items.append(URLQueryItem(name: "q", value: text))
        }
        if let spotType = spotType, !spotType.isEmpty, spotType != "all" {
            items.append(URLQueryItem(name: "type", value: spotType))
        }
        if !tags.isEmpty {
            items.append(URLQueryItem(name: "tags", value: tags.joined(separator: ",")))
        }
        return items
    }

    var isEmpty: Bool {
        queryItems.isEmpty
    }
}
