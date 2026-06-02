import CryptoKit
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

enum FrequencyCurveConstants {
    static let minFrequencyHz = 8.175798915643707
    static let maxFrequencyHz = 12_543.853951415975
    static let minGainDB = -36.0
    static let maxGainDB = 12.0
    static let maxPoints = 8
    static let boundaryToleranceHz = 1e-6
}

struct FrequencyCurvePoint: Identifiable, Codable, Equatable {
    var id: UUID
    var frequencyHz: Double
    var gainDB: Double

    init(id: UUID = UUID(), frequencyHz: Double, gainDB: Double) {
        self.id = id
        self.frequencyHz = frequencyHz
        self.gainDB = gainDB
    }

    static func defaultAnchors() -> [FrequencyCurvePoint] {
        [
            FrequencyCurvePoint(
                frequencyHz: FrequencyCurveConstants.minFrequencyHz,
                gainDB: 0.0
            ),
            FrequencyCurvePoint(
                frequencyHz: FrequencyCurveConstants.maxFrequencyHz,
                gainDB: 0.0
            ),
        ]
    }
}

struct WaveLayer: Identifiable, Codable, Equatable {
    var id: UUID
    var type: WaveformType
    var duty: Double
    var volume: Double
    var isFrequencyCurveEnabled: Bool
    var frequencyCurve: [FrequencyCurvePoint]

    init(
        id: UUID = UUID(),
        type: WaveformType,
        duty: Double = 0.5,
        volume: Double = 1.0,
        isFrequencyCurveEnabled: Bool = false,
        frequencyCurve: [FrequencyCurvePoint] = FrequencyCurvePoint.defaultAnchors()
    ) {
        self.id = id
        self.type = type
        self.duty = duty
        self.volume = volume
        self.isFrequencyCurveEnabled = isFrequencyCurveEnabled
        self.frequencyCurve = frequencyCurve
    }

    var exportedFrequencyCurve: [FrequencyCurvePoint] {
        guard isFrequencyCurveEnabled else {
            return []
        }

        let curve = frequencyCurve.isEmpty ? FrequencyCurvePoint.defaultAnchors() : frequencyCurve
        return curve.sorted { $0.frequencyHz < $1.frequencyHz }.map { point in
            var normalizedPoint = point
            if abs(normalizedPoint.frequencyHz - FrequencyCurveConstants.minFrequencyHz)
                <= FrequencyCurveConstants.boundaryToleranceHz {
                normalizedPoint.frequencyHz = FrequencyCurveConstants.minFrequencyHz
            }
            if abs(normalizedPoint.frequencyHz - FrequencyCurveConstants.maxFrequencyHz)
                <= FrequencyCurveConstants.boundaryToleranceHz {
                normalizedPoint.frequencyHz = FrequencyCurveConstants.maxFrequencyHz
            }
            return normalizedPoint
        }
    }

    var hasActiveFrequencyCurve: Bool {
        !exportedFrequencyCurve.isEmpty
    }

    mutating func ensureDefaultFrequencyCurve() {
        if frequencyCurve.isEmpty {
            frequencyCurve = FrequencyCurvePoint.defaultAnchors()
        }
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

struct RuntimeFrequencyCurvePointPayload: Equatable {
    let frequencyHz: Double
    let gainDB: Double
}

struct RuntimeLayerPayload: Equatable {
    let type: String
    let duty: Double
    let volume: Double
    let frequencyCurve: [RuntimeFrequencyCurvePointPayload]
}

enum WaveLayerExportSanitizer {
    static func sanitizedLayers(from layers: [WaveLayer]) -> [WaveLayer] {
        let audibleLayers = layers
            .filter { $0.volume > 0 }
            .map { layer in
                var sanitizedLayer = layer
                sanitizedLayer.ensureDefaultFrequencyCurve()
                return sanitizedLayer
            }

        if audibleLayers.isEmpty {
            return [WaveLayer(type: .pulse, duty: 0.5, volume: 1.0)]
        }

        return audibleLayers
    }
}

enum LayerPayloadEncoder {
    static func runtimePayloads(from layers: [WaveLayer]) -> [RuntimeLayerPayload] {
        WaveLayerExportSanitizer.sanitizedLayers(from: layers).map { layer in
            RuntimeLayerPayload(
                type: layer.type.rawValue,
                duty: layer.duty,
                volume: layer.volume,
                frequencyCurve: layer.exportedFrequencyCurve.map {
                    RuntimeFrequencyCurvePointPayload(
                        frequencyHz: $0.frequencyHz,
                        gainDB: $0.gainDB
                    )
                }
            )
        }
    }

    static func jsonString(for layers: [WaveLayer]) -> String {
        let payloads = runtimePayloads(from: layers)
        return "[\(payloads.map(layerJSONString).joined(separator: ","))]"
    }

    private static func layerJSONString(_ layer: RuntimeLayerPayload) -> String {
        "{\"duty\":\(numberString(layer.duty)),\"frequency_curve\":[\(layer.frequencyCurve.map(pointJSONString).joined(separator: ","))],\"type\":\"\(layer.type)\",\"volume\":\(numberString(layer.volume))}"
    }

    private static func pointJSONString(_ point: RuntimeFrequencyCurvePointPayload) -> String {
        "{\"frequency_hz\":\(numberString(point.frequencyHz)),\"gain_db\":\(numberString(point.gainDB))}"
    }

    private static func numberString(_ value: Double) -> String {
        String(value)
    }
}

enum OutputFileNameBuilder {
    static func outputSuffix(for layers: [WaveLayer]) -> String {
        let runtimePayloads = LayerPayloadEncoder.runtimePayloads(from: layers)
        let baseSuffix = runtimePayloads.count > 1 ? "mix" : runtimePayloads[0].type

        if runtimePayloads.contains(where: { !$0.frequencyCurve.isEmpty }) {
            let payloadJSON = LayerPayloadEncoder.jsonString(for: layers)
            return "\(baseSuffix)_\(curvePayloadHash(for: payloadJSON))"
        }

        return baseSuffix
    }

    static func outputURL(for inputURL: URL, in directory: URL, layers: [WaveLayer]) -> URL {
        let filename = inputURL.deletingPathExtension().lastPathComponent
        return directory.appendingPathComponent("\(filename)_\(outputSuffix(for: layers)).wav")
    }

    private static func curvePayloadHash(for payloadJSON: String) -> String {
        let digest = Insecure.SHA1.hash(data: Data(payloadJSON.utf8))
        return digest.prefix(4).map { String(format: "%02x", $0) }.joined()
    }
}
