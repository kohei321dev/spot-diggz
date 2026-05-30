import SwiftUI

/// A tab bar hosting the primary sections of the app.
struct RootTabView: View {
    @EnvironmentObject var appState: SdzAppState

    var body: some View {
        TabView(selection: $appState.selectedTab) {

            HomeView()
                .tabItem {
                    Label("スポット", systemImage: "map")
                }
                .tag(SdzTab.spots)
            SpotListView()
                .tabItem {
                    Label("一覧", systemImage: "list.bullet.rectangle")
                }
                .tag(SdzTab.list)
            ProfileView()
                .tabItem {
                    Label("設定", systemImage: "gearshape")
                }
                .tag(SdzTab.settings)
        }
        .tint(Color.sdzStreet)
        .sheet(isPresented: $appState.isPostComposerPresented) {
            PostView()
        }
        .confirmationDialog(
            "位置情報の反映先を選択",
            isPresented: $appState.isShareSelectionPromptVisible,
            titleVisibility: .visible
        ) {
            Button("投稿に使う") {
                appState.applyShareSelectionToPost()
            }
            Button("地図で見る") {
                appState.applyShareSelectionToMap()
            }
            Button("キャンセル", role: .cancel) {
                appState.clearShareSelection()
            }
        } message: {
            if let name = appState.pendingShareSelectionName, !name.isEmpty {
                Text(name)
            } else {
                Text("共有した位置情報をどこに反映しますか？")
            }
        }
    }
}

#if DEBUG
struct RootTabView_Previews: PreviewProvider {
    static var previews: some View {
        RootTabView()
            .environmentObject(SdzAppState())
    }
}
#endif
