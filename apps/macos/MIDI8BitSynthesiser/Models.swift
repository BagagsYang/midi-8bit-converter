import Foundation

enum WaveformType: String, CaseIterable, Codable, Identifiable {
    case pulse
    case sine
    case sawtooth
    case triangle

    var id: String { rawValue }

    var displayName: String {
        switch self {
        case .pulse:
            return "Square / Pulse"
        case .sine:
            return "Sine"
        case .sawtooth:
            return "Sawtooth"
        case .triangle:
            return "Triangle"
        }
    }

    var compactLabel: String {
        switch self {
        case .pulse:
            return "Pulse"
        case .sine:
            return "Sine"
        case .sawtooth:
            return "Saw"
        case .triangle:
            return "Tri"
        }
    }

    var symbolName: String {
        switch self {
        case .pulse:
            return "square.fill"
        case .sine:
            return "circle.fill"
        case .sawtooth:
            return "slash.forward"
        case .triangle:
            return "triangle.fill"
        }
    }
}

struct WaveLayer: Identifiable, Codable, Equatable {
    var id: UUID
    var type: WaveformType
    var duty: Double
    var volume: Double

    init(id: UUID = UUID(), type: WaveformType, duty: Double = 0.5, volume: Double = 1.0) {
        self.id = id
        self.type = type
        self.duty = duty
        self.volume = volume
    }
}

enum JobStatus: Equatable {
    case queued
    case processing
    case completed(URL)
    case failed(String)

    var label: String {
        switch self {
        case .queued:
            return "Queued"
        case .processing:
            return "Processing"
        case .completed:
            return "Completed"
        case .failed:
            return "Failed"
        }
    }
}

struct ConversionJob: Identifiable, Equatable {
    var id: UUID
    var inputURL: URL
    var outputDirectoryURL: URL?
    var sampleRate: Int
    var layers: [WaveLayer]
    var status: JobStatus

    init(
        id: UUID = UUID(),
        inputURL: URL,
        outputDirectoryURL: URL? = nil,
        sampleRate: Int = 48_000,
        layers: [WaveLayer] = [.init(type: .pulse, duty: 0.5, volume: 1.0)],
        status: JobStatus = .queued
    ) {
        self.id = id
        self.inputURL = inputURL
        self.outputDirectoryURL = outputDirectoryURL
        self.sampleRate = sampleRate
        self.layers = layers
        self.status = status
    }
}
