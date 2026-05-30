import SwiftUI

/// A simple authentication screen that lets the user sign in or create an account.
struct AuthView: View {
    @EnvironmentObject var appState: SdzAppState

    @State private var email: String = ""
    @State private var password: String = ""
    @State private var confirmPassword: String = ""
    @State private var isSignUp: Bool = false
    @State private var isBusy: Bool = false
    @State private var errorMessage: String?

    var body: some View {
        VStack {
            Spacer()
            Text("SpotDiggz")
                .font(SdzTypography.display)

            TextField("メールアドレス", text: $email)
                .textContentType(.emailAddress)
                .keyboardType(.emailAddress)
                .textInputAutocapitalization(.never)
                .disabled(isBusy)
                .padding()
                .background(Color.sdzBgSecondary)
                .cornerRadius(SdzRadius.sm)

            SecureField("パスワード", text: $password)
                .textContentType(isSignUp ? .newPassword : .password)
                .disabled(isBusy)
                .padding()
                .background(Color.sdzBgSecondary)
                .cornerRadius(SdzRadius.sm)

            if isSignUp {
                SecureField("パスワード（確認）", text: $confirmPassword)
                    .textContentType(.newPassword)
                    .disabled(isBusy)
                    .padding()
                    .background(Color.sdzBgSecondary)
                    .cornerRadius(SdzRadius.sm)
            }

            if let errorMessage = errorMessage {
                Text(errorMessage)
                    .foregroundColor(.sdzError)
                    .padding(.top, SdzSpacing.xs)
            }

            Button(action: {
                handlePrimaryAction()
            }) {
                Text(isBusy ? "処理中..." : (isSignUp ? "新規登録" : "ログイン"))
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.sdzStreet)
                    .foregroundColor(.white)
                    .cornerRadius(SdzRadius.sm)
            }
            .padding(.top, SdzSpacing.lg)
            .disabled(isBusy)

            Button(action: {
                handleGoogleSignIn()
            }) {
                Label("Googleで続ける", systemImage: "g.circle.fill")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.sdzBgSecondary)
                    .foregroundColor(.sdzTextPrimary)
                    .cornerRadius(SdzRadius.sm)
            }
            .padding(.top, SdzSpacing.sm)
            .disabled(isBusy)

            Button(action: {
                withAnimation(SdzAnimation.fast) {
                    isSignUp.toggle()
                    errorMessage = nil
                }
            }) {
                Text(isSignUp ? "ログインに戻る" : "新規登録")
                    .foregroundColor(.sdzStreet)
                    .padding(.top, SdzSpacing.sm)
            }
            .disabled(isBusy)
            Spacer()
        }
        .padding()
    }

    private func handlePrimaryAction() {
        errorMessage = nil
        let trimmedEmail = email.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !trimmedEmail.isEmpty else {
            errorMessage = "メールアドレスを入力してください。"
            return
        }
        guard !password.isEmpty else {
            errorMessage = "パスワードを入力してください。"
            return
        }
        if isSignUp && password != confirmPassword {
            errorMessage = "パスワードが一致しません。"
            return
        }

        isBusy = true
        Task {
            do {
                if isSignUp {
                    try await appState.signUpWithEmail(email: trimmedEmail, password: password)
                } else {
                    try await appState.signInWithEmail(email: trimmedEmail, password: password)
                }
            } catch {
                let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
                errorMessage = message
            }
            isBusy = false
        }
    }

    private func handleGoogleSignIn() {
        errorMessage = nil
        isBusy = true
        Task {
            do {
                try await appState.signInWithGoogle()
            } catch {
                let message = (error as? LocalizedError)?.errorDescription ?? error.localizedDescription
                errorMessage = message
            }
            isBusy = false
        }
    }
}

#if DEBUG
struct AuthView_Previews: PreviewProvider {
    static var previews: some View {
        AuthView()
            .environmentObject(SdzAppState())
    }
}
#endif
