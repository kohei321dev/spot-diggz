import Foundation
import MapKit
import UIKit

enum SdzMapNavigationMode {
    case walk
    case transit
    case drive

    var googleValue: String {
        switch self {
        case .walk:
            return "walking"
        case .transit:
            return "transit"
        case .drive:
            return "driving"
        }
    }

    var appleValue: String {
        switch self {
        case .walk:
            return MKLaunchOptionsDirectionsModeWalking
        case .transit:
            return MKLaunchOptionsDirectionsModeTransit
        case .drive:
            return MKLaunchOptionsDirectionsModeDriving
        }
    }
}

@MainActor
enum SdzMapNavigator {
    static func openGoogleMaps(
        destination: CLLocationCoordinate2D,
        waypoints: [CLLocationCoordinate2D] = [],
        mode: SdzMapNavigationMode
    ) {
        var components = URLComponents(string: "https://www.google.com/maps/dir/")
        var items = [
            URLQueryItem(name: "api", value: "1"),
            URLQueryItem(name: "destination", value: formatCoordinate(destination)),
            URLQueryItem(name: "travelmode", value: mode.googleValue),
        ]

        if !waypoints.isEmpty {
            let value = waypoints.map(formatCoordinate).joined(separator: "|")
            items.append(URLQueryItem(name: "waypoints", value: value))
        }

        components?.queryItems = items
        guard let url = components?.url else {
            return
        }
        UIApplication.shared.open(url, options: [:], completionHandler: nil)
    }

    static func openAppleMaps(destination: CLLocationCoordinate2D, mode: SdzMapNavigationMode) {
        let item: MKMapItem
        if #available(iOS 26.0, *) {
            let location = CLLocation(latitude: destination.latitude, longitude: destination.longitude)
            item = MKMapItem(location: location, address: nil)
        } else {
            let placemark = MKPlacemark(coordinate: destination)
            item = MKMapItem(placemark: placemark)
        }
        item.name = "目的地"
        MKMapItem.openMaps(
            with: [item],
            launchOptions: [MKLaunchOptionsDirectionsModeKey: mode.appleValue]
        )
    }

    private nonisolated static func formatCoordinate(_ coordinate: CLLocationCoordinate2D) -> String {
        "\(coordinate.latitude),\(coordinate.longitude)"
    }
}
