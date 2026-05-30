import SwiftUI

/// A design system divider using the `sdzDivider` color token.
struct SdzDividerView: View {
    var body: some View {
        Divider().overlay(Color.sdzDivider)
    }
}
