import SwiftUI
import MapKit

/// A balloon-shaped icon container for map pins.
struct SdzBalloonIcon<Content: View>: View {
    let color: Color
    let diameter: CGFloat
    let tailWidth: CGFloat
    let tailHeight: CGFloat
    let content: Content

    init(
        color: Color,
        diameter: CGFloat = 36,
        tailWidth: CGFloat = 12,
        tailHeight: CGFloat = 7,
        @ViewBuilder content: () -> Content
    ) {
        self.color = color
        self.diameter = diameter
        self.tailWidth = tailWidth
        self.tailHeight = tailHeight
        self.content = content()
    }

    var body: some View {
        VStack(spacing: 0) {
            ZStack {
                Circle()
                    .fill(color)
                    .frame(width: diameter, height: diameter)
                    .sdzShadow(.sm)
                content
            }
            SdzTrianglePointer()
                .fill(color)
                .frame(width: tailWidth, height: tailHeight)
                .offset(y: -1)
        }
    }
}

/// A downward-pointing triangle shape for pin tails.
struct SdzTrianglePointer: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        path.move(to: CGPoint(x: rect.midX, y: rect.maxY))
        path.addLine(to: CGPoint(x: rect.minX, y: rect.minY))
        path.addLine(to: CGPoint(x: rect.maxX, y: rect.minY))
        path.closeSubpath()
        return path
    }
}

/// A map pin view for spots with focused/unfocused and approved states.
struct SdzMapPinView: View {
    let spot: SdzSpot
    let isFocused: Bool

    var body: some View {
        let pinColor = spot.sdzPinColor
        let isApproved = spot.approvalStatus == .approved

        VStack(spacing: SdzSpacing.xs) {
            VStack(spacing: SdzSpacing.xxs) {
                if isFocused {
                    ZStack(alignment: .topTrailing) {
                        SdzBalloonIcon(color: pinColor, diameter: 48, tailWidth: 14, tailHeight: 8) {
                            Image(spot.sdzIsPark ? "SkateparkIcon" : "StreetIcon")
                                .renderingMode(.template)
                                .resizable()
                                .scaledToFit()
                                .frame(width: 24, height: 24)
                                .foregroundColor(.white)
                        }
                        if isApproved {
                            approvedBadge(size: 16, fontSize: .caption)
                                .offset(x: 12, y: -12)
                        }
                    }
                } else {
                    ZStack(alignment: .topTrailing) {
                        Image(spot.sdzIsPark ? "SkateparkIcon" : "StreetIcon")
                            .renderingMode(.template)
                            .resizable()
                            .scaledToFit()
                            .frame(width: 20, height: 20)
                            .foregroundColor(pinColor)
                            .sdzShadow(.sm)
                        if isApproved {
                            approvedBadge(size: 14, fontSize: .caption2)
                                .offset(x: 6, y: -6)
                        }
                    }
                }
                ZStack {
                    Circle()
                        .fill(pinColor)
                        .frame(
                            width: isFocused ? 10 : 8,
                            height: isFocused ? 10 : 8
                        )
                    if !isApproved {
                        Circle()
                            .stroke(
                                Color.white.opacity(0.8),
                                style: StrokeStyle(lineWidth: 2, dash: [3, 2])
                            )
                            .frame(
                                width: isFocused ? 16 : 14,
                                height: isFocused ? 16 : 14
                            )
                    }
                }
            }
            if isFocused {
                Text(spot.name)
                    .font(.caption.bold())
                    .foregroundColor(.primary)
                    .lineLimit(1)
                    .sdzShadow(.sm)
            }
        }
        .animation(.spring(response: 0.25, dampingFraction: 0.8), value: isFocused)
    }

    private func approvedBadge(size: CGFloat, fontSize: Font) -> some View {
        Image(systemName: "checkmark.seal.fill")
            .font(fontSize)
            .foregroundColor(.white)
            .background(
                Circle()
                    .fill(spot.sdzPinColor)
                    .frame(width: size, height: size)
            )
    }
}

/// A cluster bubble view showing the count of grouped spots.
struct SdzClusterBubble: View {
    let count: Int
    let color: Color

    var body: some View {
        SdzBalloonIcon(color: color) {
            Text("＋\(count)")
                .font(.caption.bold())
                .foregroundColor(.white)
        }
    }
}

#if DEBUG
struct SdzMapPin_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 24) {
            SdzMapPinView(spot: .sample(id: "1", name: "テストパーク"), isFocused: false)
            SdzMapPinView(spot: .sample(id: "2", name: "テストパーク"), isFocused: true)
            SdzClusterBubble(count: 5, color: .sdzPark)
        }
        .padding()
        .previewLayout(.sizeThatFits)
    }
}
#endif
