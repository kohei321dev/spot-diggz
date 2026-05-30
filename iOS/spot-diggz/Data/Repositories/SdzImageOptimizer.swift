import UIKit

struct SdzImagePayload {
    let data: Data
    let contentType: String
}

enum SdzImageOptimizer {
    private static let maxDimension: CGFloat = 1080
    private static let jpegQuality: CGFloat = 0.82

    static func optimize(_ image: UIImage) -> SdzImagePayload? {
        let resized = resize(image, maxDimension: maxDimension)
        if hasAlpha(resized) {
            guard let data = resized.pngData() else {
                return nil
            }
            return SdzImagePayload(data: data, contentType: "image/png")
        }
        guard let data = resized.jpegData(compressionQuality: jpegQuality) else {
            return nil
        }
        return SdzImagePayload(data: data, contentType: "image/jpeg")
    }

    private static func resize(_ image: UIImage, maxDimension: CGFloat) -> UIImage {
        let size = image.size
        let maxSide = max(size.width, size.height)
        guard maxSide > maxDimension else {
            return image
        }
        let scale = maxDimension / maxSide
        let newSize = CGSize(width: size.width * scale, height: size.height * scale)
        let renderer = UIGraphicsImageRenderer(size: newSize)
        return renderer.image { _ in
            image.draw(in: CGRect(origin: .zero, size: newSize))
        }
    }

    private static func hasAlpha(_ image: UIImage) -> Bool {
        guard let alpha = image.cgImage?.alphaInfo else {
            return false
        }
        switch alpha {
        case .first, .last, .premultipliedFirst, .premultipliedLast:
            return true
        default:
            return false
        }
    }
}
