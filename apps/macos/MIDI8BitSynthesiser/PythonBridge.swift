import Foundation

enum PythonBridgeError: LocalizedError {
    case helperMissing
    case launchFailed(String)
    case processFailed(code: Int32, message: String)
    case invalidLayerPayload(String)

    var errorDescription: String? {
        switch self {
        case .helperMissing:
            return "Bundled converter helper is missing. Build the app from Xcode so the Python helper is embedded in the app bundle."
        case .launchFailed(let message):
            return "Could not launch the converter helper: \(message)"
        case .processFailed(let code, let message):
            let trimmedMessage = message.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmedMessage.isEmpty
                ? "The converter helper exited with code \(code)."
                : trimmedMessage
        case .invalidLayerPayload(let message):
            return "Could not encode waveform layers: \(message)"
        }
    }
}

struct PythonBridge {
    static func convert(
        inputURL: URL,
        outputURL: URL,
        sampleRate: Int,
        layers: [WaveLayer]
    ) async throws {
        let helperURL = try helperExecutableURL()
        let encoder = JSONEncoder()

        let payload: Data
        do {
            payload = try encoder.encode(
                layers.map {
                    EncodedWaveLayer(type: $0.type.rawValue, duty: $0.duty, volume: $0.volume)
                }
            )
        } catch {
            throw PythonBridgeError.invalidLayerPayload(error.localizedDescription)
        }

        let jsonString = String(decoding: payload, as: UTF8.self)
        _ = try await ProcessRunner.run(
            executableURL: helperURL,
            arguments: [
                inputURL.path,
                outputURL.path,
                "--rate",
                String(sampleRate),
                "--layers-json",
                jsonString,
            ]
        )
    }

    private static func helperExecutableURL() throws -> URL {
        guard let bundledURL = Bundle.main.url(
            forResource: "midi_to_wave_helper",
            withExtension: nil,
            subdirectory: "python"
        ) else {
            throw PythonBridgeError.helperMissing
        }

        return bundledURL
    }
}

private struct EncodedWaveLayer: Encodable {
    let type: String
    let duty: Double
    let volume: Double
}

private struct ProcessResult {
    let stdout: String
    let stderr: String
    let exitCode: Int32
}

private enum ProcessRunner {
    static func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult {
        try await withCheckedThrowingContinuation { continuation in
            let process = Process()
            let stdoutPipe = Pipe()
            let stderrPipe = Pipe()

            process.executableURL = executableURL
            process.arguments = arguments
            process.standardOutput = stdoutPipe
            process.standardError = stderrPipe

            process.terminationHandler = { process in
                let stdoutData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
                let stderrData = stderrPipe.fileHandleForReading.readDataToEndOfFile()
                let result = ProcessResult(
                    stdout: String(decoding: stdoutData, as: UTF8.self),
                    stderr: String(decoding: stderrData, as: UTF8.self),
                    exitCode: process.terminationStatus
                )

                if process.terminationStatus == 0 {
                    continuation.resume(returning: result)
                } else {
                    let message = result.stderr.isEmpty ? result.stdout : result.stderr
                    continuation.resume(
                        throwing: PythonBridgeError.processFailed(
                            code: result.exitCode,
                            message: message
                        )
                    )
                }
            }

            do {
                try process.run()
            } catch {
                continuation.resume(
                    throwing: PythonBridgeError.launchFailed(error.localizedDescription)
                )
            }
        }
    }
}
