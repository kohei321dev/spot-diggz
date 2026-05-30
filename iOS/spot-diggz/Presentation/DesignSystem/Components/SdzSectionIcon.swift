import SwiftUI

/// Skate park section types with SF Symbols and Japanese labels.
enum SdzParkSection: String, CaseIterable, Identifiable, Codable {
    case bowl
    case halfPipe
    case miniRamp
    case quarterPipe
    case funBox
    case ledge
    case rail
    case stairs
    case handrail
    case bank
    case gap
    case manual
    case flatBar
    case kicker
    case pyramid

    var id: String { rawValue }

    var label: String {
        switch self {
        case .bowl: return "ボウル"
        case .halfPipe: return "ハーフパイプ"
        case .miniRamp: return "ミニランプ"
        case .quarterPipe: return "クォーターパイプ"
        case .funBox: return "ファンボックス"
        case .ledge: return "レッジ"
        case .rail: return "レール"
        case .stairs: return "ステアー"
        case .handrail: return "ハンドレール"
        case .bank: return "バンク"
        case .gap: return "ギャップ"
        case .manual: return "マニュアル台"
        case .flatBar: return "フラットバー"
        case .kicker: return "キッカー"
        case .pyramid: return "ピラミッド"
        }
    }

    var symbolName: String {
        switch self {
        case .bowl: return "circle.bottomhalf.filled"
        case .halfPipe: return "arrow.left.and.right"
        case .miniRamp: return "arrow.up.left.and.arrow.down.right"
        case .quarterPipe: return "arrow.up.right"
        case .funBox: return "cube"
        case .ledge: return "rectangle.split.3x1"
        case .rail: return "line.diagonal"
        case .stairs: return "stairs"
        case .handrail: return "line.diagonal.arrow"
        case .bank: return "triangle"
        case .gap: return "arrowtriangle.down"
        case .manual: return "rectangle"
        case .flatBar: return "minus"
        case .kicker: return "arrow.up.forward"
        case .pyramid: return "triangle.fill"
        }
    }
}

/// A single section icon with label.
struct SdzSectionIconView: View {
    let section: SdzParkSection
    var isCompact: Bool = false

    var body: some View {
        VStack(spacing: SdzSpacing.xs) {
            Image(systemName: section.symbolName)
                .font(isCompact ? SdzTypography.caption1 : SdzTypography.headline)
                .foregroundColor(.sdzPark)
                .frame(
                    width: isCompact ? 28 : 40,
                    height: isCompact ? 28 : 40
                )
                .background(Color.sdzPark.opacity(0.1))
                .clipShape(RoundedRectangle(cornerRadius: SdzRadius.sm, style: .continuous))
            Text(section.label)
                .font(isCompact ? SdzTypography.caption2 : SdzTypography.caption1)
                .foregroundColor(.sdzTextSecondary)
                .lineLimit(1)
        }
    }
}

/// A grid displaying available park sections.
struct SdzSectionGrid: View {
    let sections: [SdzParkSection]
    var columns: Int = 4

    var body: some View {
        LazyVGrid(
            columns: Array(repeating: GridItem(.flexible(), spacing: SdzSpacing.sm), count: columns),
            spacing: SdzSpacing.md
        ) {
            ForEach(sections) { section in
                SdzSectionIconView(section: section, isCompact: true)
            }
        }
    }
}

#if DEBUG
struct SdzSectionIcon_Previews: PreviewProvider {
    static var previews: some View {
        VStack(spacing: 16) {
            SdzSectionIconView(section: .bowl)
            SdzSectionGrid(sections: Array(SdzParkSection.allCases.prefix(8)))
        }
        .padding()
        .previewLayout(.sizeThatFits)
    }
}
#endif
