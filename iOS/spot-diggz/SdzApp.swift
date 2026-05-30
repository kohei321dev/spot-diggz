import SwiftUI

/// The entry point for the SpotDiggz application.
@main
struct SdzApp: App {
    /// Global application state.
    @StateObject private var appState = SdzAppState()
    @State private var themeManager = SdzThemeManager()
    @Environment(\.scenePhase) private var scenePhase

    init() {
        _ = SdzAuthService.shared.configureIfNeeded()
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if appState.isAuthLoading {
                    ProgressView("認証中...")
                } else if appState.isAuthenticated {
                    RootTabView()
                } else {
                    AuthView()
                }
            }
            .environmentObject(appState)
            .environment(themeManager)
            .preferredColorScheme(themeManager.resolvedColorScheme)
            .task {
                await appState.restoreSession()
                appState.consumeSharedPayloadIfNeeded()
            }
            .onOpenURL { url in
                if SdzAuthService.shared.handleOpenUrl(url) {
                    return
                }
                _ = appState.handleIncomingUrl(url)
            }
            .onContinueUserActivity(NSUserActivityTypeBrowsingWeb) { activity in
                if let url = activity.webpageURL {
                    _ = appState.handleIncomingUrl(url)
                }
            }
            .onChange(of: scenePhase) { _, newPhase in
                if newPhase == .active {
                    appState.consumeSharedPayloadIfNeeded()
                }
            }
        }
    }
}
