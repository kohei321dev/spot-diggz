import SwiftUI
import MapKit

/// Displays a map with pins representing nearby skate spots.
struct MapScreen: View {
    @EnvironmentObject var appState: SdzAppState
    @State private var region = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 35.0, longitude: 135.0),
        span: MKCoordinateSpan(latitudeDelta: 0.05, longitudeDelta: 0.05)
    )
    @State private var spots: [SdzSpot] = []
    @State private var selectedSpot: SdzSpot?

    var body: some View {
        Group {
            if #available(iOS 16.0, *) {
                NavigationStack {
                    mapContent
                        .ignoresSafeArea(edges: .top)
                        .navigationTitle("マップ")
                        .onAppear {
                            fetchSpots()
                        }
                        .sheet(item: $selectedSpot) { spot in
                            if #available(iOS 16.0, *) {
                                NavigationStack {
                                    SpotDetailView(spot: spot)
                                }
                            } else {
                                NavigationView {
                                    SpotDetailView(spot: spot)
                                }
                            }
                        }
                }
            } else {
                NavigationView {
                    mapContent
                        .ignoresSafeArea(edges: .top)
                        .navigationTitle("マップ")
                        .onAppear {
                            fetchSpots()
                        }
                        .sheet(item: $selectedSpot) { spot in
                            NavigationView {
                                SpotDetailView(spot: spot)
                            }
                        }
                }
            }
        }
    }

    // MARK: - Map Content split by iOS version to avoid deprecation warnings
    @ViewBuilder
    private var mapContent: some View {
        if #available(iOS 17.0, *) {
            // iOS 17+: Use MapContentBuilder-based API
            Map(initialPosition: .region(region)) {
                ForEach(annotationItems) { annotation in
                    Annotation("", coordinate: annotation.coordinate) {
                        Button(action: {
                            selectedSpot = annotation.spot
                        }) {
                            Image(systemName: "mappin.circle.fill")
                                .font(.title)
                                .foregroundColor(annotation.spot.sdzPinColor)
                        }
                    }
                }
            }
        } else {
            // iOS 16 and earlier: Use legacy API (not compiled on iOS 17+, so no warnings)
            Map(coordinateRegion: $region, annotationItems: annotationItems) { annotation in
                MapAnnotation(coordinate: annotation.coordinate) {
                    Button(action: {
                        selectedSpot = annotation.spot
                    }) {
                        Image(systemName: "mappin.circle.fill")
                            .font(.title)
                            .foregroundColor(annotation.spot.sdzPinColor)
                    }
                }
            }
        }
    }

    private struct AnnotationItem: Identifiable {
        let id: String
        let coordinate: CLLocationCoordinate2D
        let spot: SdzSpot
    }

    private var annotationItems: [AnnotationItem] {
        spots.compactMap { spot in
            if let location = spot.location {
                return AnnotationItem(
                    id: spot.id,
                    coordinate: CLLocationCoordinate2D(latitude: location.lat, longitude: location.lng),
                    spot: spot
                )
            }
            return nil
        }
    }

    private func fetchSpots() {
        let apiClient = SdzApiClient(environment: appState.environment, idToken: appState.idToken)
        Task {
            do {
                let result = try await apiClient.fetchSpots(includeAuth: true)
                await MainActor.run {
                    self.spots = result
                }
            } catch {
                // Map screen keeps the last successful data; no UI error yet.
                print("Failed to fetch spots: \(error.localizedDescription)")
            }
        }
    }
}

#if DEBUG
struct MapScreen_Previews: PreviewProvider {
    static var previews: some View {
        MapScreen()
            .environmentObject(SdzAppState())
    }
}
#endif
