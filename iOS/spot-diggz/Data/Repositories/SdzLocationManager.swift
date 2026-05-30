import Foundation
import Combine
import CoreLocation

@MainActor
final class SdzLocationManager: NSObject, ObservableObject, CLLocationManagerDelegate {
    @Published var authorizationStatus: CLAuthorizationStatus
    @Published var currentCoordinate: CLLocationCoordinate2D?
    @Published var lastErrorMessage: String?

    private let manager: CLLocationManager

    override init() {
        manager = CLLocationManager()
        authorizationStatus = manager.authorizationStatus
        super.init()
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyBest
    }

    func requestWhenInUseAuthorization() {
        manager.requestWhenInUseAuthorization()
    }

    func requestCurrentLocation() {
        lastErrorMessage = nil
        switch authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            if let cachedCoordinate = manager.location?.coordinate {
                currentCoordinate = cachedCoordinate
            }
            manager.requestLocation()
        case .notDetermined:
            requestWhenInUseAuthorization()
        case .denied, .restricted:
            lastErrorMessage = "位置情報の利用が許可されていません。設定アプリで許可してください。"
        @unknown default:
            manager.requestLocation()
        }
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        authorizationStatus = manager.authorizationStatus
        switch authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            requestCurrentLocation()
        case .denied, .restricted:
            lastErrorMessage = "位置情報の利用が許可されていません。設定アプリで許可してください。"
        default:
            break
        }
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        currentCoordinate = locations.last?.coordinate
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        if let locationError = error as? CLError, locationError.code == .locationUnknown {
            return
        }
        lastErrorMessage = error.localizedDescription
    }
}
