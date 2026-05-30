import SwiftUI

/// Displays a detailed list of favorite spots.
struct MyListDetailView: View {
    @EnvironmentObject var appState: SdzAppState

    var body: some View {
        List {
            if appState.isFavoritesLoading {
                ProgressView("読み込み中...")
            } else if let errorMessage = appState.favoritesErrorMessage {
                Text(errorMessage)
                    .foregroundColor(.red)
            } else if appState.favoriteSpots.isEmpty {
                Text("お気に入りはありません")
                    .foregroundColor(.secondary)
            } else {
                ForEach(appState.favoriteSpots) { spot in
                    NavigationLink(destination: SpotDetailView(spot: spot)) {
                        SpotCardView(spot: spot)
                    }
                }
            }
        }
        .navigationTitle("お気に入り一覧")
        .task {
            await appState.refreshFavorites()
        }
    }
}

#if DEBUG
struct MyListDetailView_Previews: PreviewProvider {
    static var previews: some View {
        NavigationView {
            MyListDetailView()
                .environmentObject(SdzAppState())
        }
    }
}
#endif
