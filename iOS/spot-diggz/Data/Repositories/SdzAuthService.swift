import Foundation

#if canImport(FirebaseCore)
import FirebaseCore
#endif
#if canImport(FirebaseAuth)
import FirebaseAuth
#endif
#if canImport(GoogleSignIn)
import GoogleSignIn
#endif
#if canImport(UIKit)
import UIKit
#endif

struct SdzAuthSession {
    let idToken: String
    let userId: String
    let displayName: String?
    let email: String?
    let providerIds: [String]
}

enum SdzAuthServiceError: LocalizedError {
    case missingDependencies
    case missingConfiguration
    case missingPresenter
    case signInFailed(String)
    case signOutFailed

    var errorDescription: String? {
        switch self {
        case .missingDependencies:
            return "Firebase/GoogleSignInの設定が未完了です。"
        case .missingConfiguration:
            return "GoogleService-Info.plistが見つかりません。"
        case .missingPresenter:
            return "ログイン画面の表示準備に失敗しました。"
        case let .signInFailed(message):
            return "ログインに失敗しました: \(message)"
        case .signOutFailed:
            return "ログアウトに失敗しました。"
        }
    }
}

@MainActor
final class SdzAuthService {
    static let shared = SdzAuthService()

    private init() {}

    @discardableResult
    func configureIfNeeded() -> Bool {
        #if canImport(FirebaseCore)
        if FirebaseApp.app() != nil {
            return true
        }
        guard let filePath = Bundle.main.path(forResource: "GoogleService-Info", ofType: "plist"),
              let options = FirebaseOptions(contentsOfFile: filePath) else {
            return false
        }
        FirebaseApp.configure(options: options)
        return true
        #else
        return false
        #endif
    }

    func currentSession() async -> SdzAuthSession? {
        #if canImport(FirebaseAuth)
        guard let user = Auth.auth().currentUser else {
            return nil
        }
        do {
            let token = try await fetchIdToken(for: user)
            return buildSession(user: user, token: token)
        } catch {
            return nil
        }
        #else
        return nil
        #endif
    }

    func signInWithEmail(email: String, password: String) async throws -> SdzAuthSession {
        #if canImport(FirebaseAuth)
        guard configureIfNeeded() else {
            throw SdzAuthServiceError.missingConfiguration
        }
        do {
            let result = try await signInWithEmailInternal(email: email, password: password)
            let token = try await fetchIdToken(for: result.user)
            return buildSession(user: result.user, token: token)
        } catch {
            throw mapAuthError(error)
        }
        #else
        throw SdzAuthServiceError.missingDependencies
        #endif
    }

    func signUpWithEmail(email: String, password: String) async throws -> SdzAuthSession {
        #if canImport(FirebaseAuth)
        guard configureIfNeeded() else {
            throw SdzAuthServiceError.missingConfiguration
        }
        do {
            let result = try await createUserInternal(email: email, password: password)
            let token = try await fetchIdToken(for: result.user)
            return buildSession(user: result.user, token: token)
        } catch {
            throw mapAuthError(error)
        }
        #else
        throw SdzAuthServiceError.missingDependencies
        #endif
    }

    func signInWithGoogle() async throws -> SdzAuthSession {
        #if canImport(FirebaseAuth) && canImport(GoogleSignIn)
        guard configureIfNeeded() else {
            throw SdzAuthServiceError.missingConfiguration
        }
        guard let presenting = SdzViewControllerResolver.topViewController() else {
            throw SdzAuthServiceError.missingPresenter
        }
        guard let clientID = FirebaseApp.app()?.options.clientID else {
            throw SdzAuthServiceError.missingConfiguration
        }

        let configuration = GIDConfiguration(clientID: clientID)
        GIDSignIn.sharedInstance.configuration = configuration

        do {
            let result = try await signInWithGoogleInternal(presenting: presenting)
            let credential = GoogleAuthProvider.credential(
                withIDToken: result.idToken,
                accessToken: result.accessToken
            )
            let authResult = try await signInWithCredentialInternal(credential)
            let token = try await fetchIdToken(for: authResult.user)
            return buildSession(user: authResult.user, token: token)
        } catch {
            throw mapAuthError(error)
        }
        #else
        throw SdzAuthServiceError.missingDependencies
        #endif
    }

    func signOut() throws {
        #if canImport(FirebaseAuth)
        do {
            try Auth.auth().signOut()
        } catch {
            throw SdzAuthServiceError.signOutFailed
        }
        #else
        throw SdzAuthServiceError.missingDependencies
        #endif
    }

    func handleOpenUrl(_ url: URL) -> Bool {
        #if canImport(GoogleSignIn)
        return GIDSignIn.sharedInstance.handle(url)
        #else
        return false
        #endif
    }

    // MARK: - Private helpers

    #if canImport(FirebaseAuth)
    private func buildSession(user: User, token: String) -> SdzAuthSession {
        let providers = user.providerData.map(\.providerID)
        return SdzAuthSession(
            idToken: token,
            userId: user.uid,
            displayName: user.displayName,
            email: user.email,
            providerIds: providers
        )
    }

    private func fetchIdToken(for user: User) async throws -> String {
        try await withCheckedThrowingContinuation { continuation in
            user.getIDToken { token, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }
                continuation.resume(returning: token ?? "")
            }
        }
    }

    private func signInWithEmailInternal(email: String, password: String) async throws -> AuthDataResult {
        try await withCheckedThrowingContinuation { continuation in
            Auth.auth().signIn(withEmail: email, password: password) { result, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }
                guard let result = result else {
                    continuation.resume(throwing: SdzAuthServiceError.signInFailed("No result"))
                    return
                }
                continuation.resume(returning: result)
            }
        }
    }

    private func createUserInternal(email: String, password: String) async throws -> AuthDataResult {
        try await withCheckedThrowingContinuation { continuation in
            Auth.auth().createUser(withEmail: email, password: password) { result, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }
                guard let result = result else {
                    continuation.resume(throwing: SdzAuthServiceError.signInFailed("No result"))
                    return
                }
                continuation.resume(returning: result)
            }
        }
    }

    private struct GoogleSignInResult {
        let idToken: String
        let accessToken: String
    }

    private func signInWithGoogleInternal(presenting: UIViewController) async throws -> GoogleSignInResult {
        #if canImport(GoogleSignIn)
        return try await withCheckedThrowingContinuation { continuation in
            GIDSignIn.sharedInstance.signIn(withPresenting: presenting) { result, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }
                guard let result = result,
                      let idToken = result.user.idToken?.tokenString else {
                    continuation.resume(throwing: SdzAuthServiceError.signInFailed("Missing token"))
                    return
                }
                let accessToken = result.user.accessToken.tokenString
                continuation.resume(returning: GoogleSignInResult(idToken: idToken, accessToken: accessToken))
            }
        }
        #else
        throw SdzAuthServiceError.missingDependencies
        #endif
    }

    private func signInWithCredentialInternal(_ credential: AuthCredential) async throws -> AuthDataResult {
        try await withCheckedThrowingContinuation { continuation in
            Auth.auth().signIn(with: credential) { result, error in
                if let error = error {
                    continuation.resume(throwing: error)
                    return
                }
                guard let result = result else {
                    continuation.resume(throwing: SdzAuthServiceError.signInFailed("No result"))
                    return
                }
                continuation.resume(returning: result)
            }
        }
    }

    private func mapAuthError(_ error: Error) -> Error {
        if let authError = error as? SdzAuthServiceError {
            return authError
        }
        return SdzAuthServiceError.signInFailed(error.localizedDescription)
    }
    #endif
}

#if canImport(UIKit)
private enum SdzViewControllerResolver {
    static func topViewController() -> UIViewController? {
        guard let scene = UIApplication.shared.connectedScenes
            .compactMap({ $0 as? UIWindowScene })
            .first(where: { $0.activationState == .foregroundActive }) else {
            return nil
        }
        guard let root = scene.windows.first(where: { $0.isKeyWindow })?.rootViewController else {
            return nil
        }
        return resolve(from: root)
    }

    private static func resolve(from controller: UIViewController) -> UIViewController {
        if let presented = controller.presentedViewController {
            return resolve(from: presented)
        }
        if let navigation = controller as? UINavigationController, let top = navigation.topViewController {
            return resolve(from: top)
        }
        if let tab = controller as? UITabBarController, let selected = tab.selectedViewController {
            return resolve(from: selected)
        }
        return controller
    }
}
#endif
