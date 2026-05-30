import UIKit
import UniformTypeIdentifiers
import MapKit

final class ShareViewController: UIViewController {
    private enum SharedDefaults {
        static let appGroupId = "group.ios-sdz-fb-dev"
        static let payloadKey = "sdz.shared-payload"
        static let debugLogFile = "sdz.share-debug.log"
    }
    private enum UniversalLinkConfig {
        static let host = "sdz-dev-api-1053202159855.asia-northeast1.run.app"
    }

    private struct SharedPayload: Codable {
        let kind: String
        let lat: Double?
        let lng: Double?
        let name: String?
        let url: String?
        let createdAt: Date
    }
    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        handleShare()
    }

    private func handleShare() {
        logDebug("handleShare: invoked")
        guard let item = extensionContext?.inputItems.first as? NSExtensionItem else {
            NSLog("ShareExtension: no inputItems")
            logDebug("handleShare: no inputItems")
            debugPresentTypeListIfNeeded(providers: []) { [weak self] in
                self?.complete()
            }
            return
        }

        let providers = item.attachments ?? []

        logDebug("handleShare: providers count=\(providers.count)")

        debugPresentTypeListIfNeeded(providers: providers) { [weak self] in
            self?.handleShareProviders(item: item, providers: providers)
        }
    }

    private func handleShareProviders(item: NSExtensionItem, providers: [NSItemProvider]) {
        if let provider = firstProvider(in: providers, conformingTo: Self.mapItemTypeIdentifier) {
            loadMapItem(from: provider)
            return
        }

        if let provider = firstProvider(in: providers, conformingTo: UTType.url.identifier) {
            provider.loadItem(forTypeIdentifier: UTType.url.identifier, options: nil) { [weak self] item, _ in
                self?.openSharedItem(item)
            }
            return
        }

        if let provider = firstProvider(in: providers, conformingTo: UTType.plainText.identifier) {
            provider.loadItem(forTypeIdentifier: UTType.plainText.identifier, options: nil) { [weak self] item, _ in
                self?.openSharedItem(item)
            }
            return
        }

        if let attributed = item.attributedContentText?.string, !attributed.isEmpty {
            open(urlString: attributed)
            return
        }

        complete()
    }

    private func debugPresentTypeListIfNeeded(providers: [NSItemProvider], completion: @escaping () -> Void) {
        completion()
    }

    private func openSharedItem(_ item: NSSecureCoding?) {
        let urlString: String?
        if let url = item as? URL {
            urlString = url.absoluteString
        } else if let value = item as? String {
            urlString = value
        } else {
            urlString = nil
        }
        open(urlString: urlString)
    }

    private func open(urlString: String?) {
        let trimmed = urlString?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else {
            complete()
            return
        }

        let host = destinationHost(for: trimmed)
        let payloadKind: String
        switch host {
        case "add-url":
            payloadKind = "officialUrl"
        default:
            payloadKind = "location"
        }
        let payload = SharedPayload(
            kind: payloadKind,
            lat: nil,
            lng: nil,
            name: nil,
            url: trimmed,
            createdAt: Date()
        )
        let success = saveSharedPayload(payload)
        promptSaved(success: success, kind: payload.kind)
    }

    private func open(mapItem: MKMapItem) {
        if #available(iOS 26.0, *) {
            let location = mapItem.location
            if CLLocationCoordinate2DIsValid(location.coordinate) {
                openLocation(
                    lat: location.coordinate.latitude,
                    lng: location.coordinate.longitude,
                    name: mapItem.name
                )
                return
            }
            if let url = mapItem.url?.absoluteString {
                open(urlString: url)
                return
            }
            complete()
            return
        }
        if let url = mapItem.url?.absoluteString {
            open(urlString: url)
            return
        }
        complete()
    }

    private func openLocation(lat: Double, lng: Double, name: String?) {
        let payload = SharedPayload(
            kind: "location",
            lat: lat,
            lng: lng,
            name: name,
            url: nil,
            createdAt: Date()
        )
        let success = saveSharedPayload(payload)
        promptSaved(success: success, kind: payload.kind)
    }

    private func destinationHost(for value: String) -> String {
        guard let url = URL(string: value) else {
            return "add-url"
        }
        if isMapUrl(url) {
            return "share-location"
        }
        return "add-url"
    }

    private func isMapUrl(_ url: URL) -> Bool {
        guard let host = url.host?.lowercased() else {
            return false
        }
        if host.contains("maps.apple.com") {
            return true
        }
        if host.contains("maps.google.com") {
            return true
        }
        if host.contains("maps.app.goo.gl") {
            return true
        }
        if host.contains("google.com") && url.path.contains("/maps") {
            return true
        }
        if host.contains("goo.gl") && url.path.contains("/maps") {
            return true
        }
        return false
    }

    private static let mapItemTypeIdentifier = "com.apple.mapkit.map-item"

    private func firstProvider(in providers: [NSItemProvider], conformingTo typeIdentifier: String) -> NSItemProvider? {
        providers.first { $0.hasItemConformingToTypeIdentifier(typeIdentifier) }
    }

    private func loadMapItem(from provider: NSItemProvider) {
        if provider.canLoadObject(ofClass: MKMapItem.self) {
            provider.loadObject(ofClass: MKMapItem.self) { [weak self] object, _ in
                if let mapItem = object as? MKMapItem {
                    self?.open(mapItem: mapItem)
                    return
                }
                self?.loadMapItemData(from: provider)
            }
            return
        }
        loadMapItemData(from: provider)
    }

    private func loadMapItemData(from provider: NSItemProvider) {
        provider.loadItem(forTypeIdentifier: Self.mapItemTypeIdentifier, options: nil) { [weak self] item, _ in
            if let mapItem = item as? MKMapItem {
                self?.open(mapItem: mapItem)
                return
            }
            if let data = item as? Data,
               let mapItem = try? NSKeyedUnarchiver.unarchivedObject(ofClass: MKMapItem.self, from: data) {
                self?.open(mapItem: mapItem)
                return
            }
            self?.openSharedItem(item)
        }
    }

    private func saveSharedPayload(_ payload: SharedPayload) -> Bool {
        guard let defaults = UserDefaults(suiteName: SharedDefaults.appGroupId) else {
            NSLog("ShareExtension: failed to access app group")
            logDebug("saveSharedPayload: failed to access app group")
            DispatchQueue.main.async { [weak self] in
                self?.presentDebugAlert(title: "App Group未接続", message: "group.ios-sdz-fb-dev にアクセスできません")
            }
            return false
        }
        do {
            let data = try JSONEncoder().encode(payload)
            defaults.set(data, forKey: SharedDefaults.payloadKey)
            logDebug("saveSharedPayload: saved kind=\(payload.kind)")
            return true
        } catch {
            NSLog("ShareExtension: failed to encode payload: %@", error.localizedDescription)
            logDebug("saveSharedPayload: encode error \(error.localizedDescription)")
            return false
        }
    }

    private func makeAppUrl(from payload: SharedPayload) -> URL? {
        if payload.kind == "officialUrl", let url = payload.url, !url.isEmpty {
            var components = URLComponents()
            components.scheme = "https"
            components.host = UniversalLinkConfig.host
            components.path = "/add-url"
            components.queryItems = [URLQueryItem(name: "url", value: url)]
            return components.url
        }
        if payload.kind == "location" {
            var components = URLComponents()
            components.scheme = "https"
            components.host = UniversalLinkConfig.host
            components.path = "/share-location"
            var items: [URLQueryItem] = []
            if let lat = payload.lat, let lng = payload.lng {
                items.append(URLQueryItem(name: "lat", value: String(lat)))
                items.append(URLQueryItem(name: "lng", value: String(lng)))
                if let name = payload.name, !name.isEmpty {
                    items.append(URLQueryItem(name: "name", value: name))
                }
            } else if let url = payload.url, !url.isEmpty {
                items.append(URLQueryItem(name: "url", value: url))
            } else {
                return nil
            }
            components.queryItems = items
            return components.url
        }
        return nil
    }

    private func openAppUrl(_ url: URL) {
        let openBlock: () -> Void = { [weak self] in
            guard let self else { return }
            self.extensionContext?.open(url, completionHandler: { success in
                if !success {
                    NSLog("ShareExtension: failed to open url=%@", url.absoluteString)
                }
                self.complete()
            })
        }
        if Thread.isMainThread {
            openBlock()
        } else {
            DispatchQueue.main.async(execute: openBlock)
        }
    }

    private func presentDebugAlert(title: String, message: String) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "OK", style: .default))
        if let presented = presentedViewController {
            presented.dismiss(animated: false) {
                self.present(alert, animated: true)
            }
        } else {
            present(alert, animated: true)
        }
    }

    private func promptSaved(success: Bool, kind: String) {
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            let title = success ? "共有情報をコピーしました" : "保存できませんでした"
            let message: String
            if success {
                switch kind {
                case "location":
                    message = "「OK」をクリックし，再度spot-diggzを開くと，位置情報がPINされます。"
                default:
                    message = "「OK」をクリックし，再度spot-diggzを開くと，URL候補が反映されます。"
                }
            } else {
                message = "App Groupの設定を確認してください。"
            }
            let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "OK", style: .default) { [weak self] _ in
                self?.complete()
            })
            if let presented = self.presentedViewController {
                presented.dismiss(animated: false) {
                    self.present(alert, animated: true)
                }
            } else {
                self.present(alert, animated: true)
            }
        }
    }

    private func complete() {
        DispatchQueue.main.async { [weak self] in
            self?.logDebug("complete: finishing extension")
            self?.extensionContext?.completeRequest(returningItems: nil, completionHandler: nil)
        }
    }

    // MARK: - Debug logging
    private func logDebug(_ message: String) {
        #if DEBUG
        NSLog("ShareExtension DEBUG: %@", message)
        guard let containerURL = FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: SharedDefaults.appGroupId) else {
            return
        }
        let logURL = containerURL.appendingPathComponent(SharedDefaults.debugLogFile)
        let timestamp = ISO8601DateFormatter().string(from: Date())
        let line = "[\(timestamp)] \(message)\n"
        if let data = line.data(using: .utf8) {
            if FileManager.default.fileExists(atPath: logURL.path) {
                if let handle = try? FileHandle(forWritingTo: logURL) {
                    handle.seekToEndOfFile()
                    handle.write(data)
                    try? handle.close()
                }
            } else {
                try? data.write(to: logURL)
            }
        }
        #endif
    }
}
