import Foundation

enum SdzTimeDefaults {
    static let weekdayStart = defaultTime(hour: 9, minute: 0)
    static let weekdayEnd = defaultTime(hour: 18, minute: 0)
    static let weekendStart = defaultTime(hour: 9, minute: 0)
    static let weekendEnd = defaultTime(hour: 18, minute: 0)

    static func time(from minutes: Int?, fallback: Date) -> Date {
        guard let minutes = minutes else {
            return fallback
        }
        let hour = minutes / 60
        let minute = minutes % 60
        return defaultTime(hour: hour, minute: minute, fallback: fallback)
    }

    private static func defaultTime(hour: Int, minute: Int, fallback: Date = Date()) -> Date {
        Calendar.current.date(bySettingHour: hour, minute: minute, second: 0, of: Date()) ?? fallback
    }
}
