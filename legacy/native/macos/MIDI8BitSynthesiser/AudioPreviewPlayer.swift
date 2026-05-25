import AVFoundation
import Foundation

enum PreviewPlaybackError: LocalizedError {
    case missingPreview(String)
    case playbackFailed(String)

    var errorDescription: String? {
        switch self {
        case .missingPreview(let name):
            return "Missing bundled preview sample: \(name).wav"
        case .playbackFailed(let message):
            return "Could not play preview: \(message)"
        }
    }
}

@MainActor
final class AudioPreviewPlayer: NSObject, ObservableObject {
    private var player: AVAudioPlayer?

    func playPreview(for layer: WaveLayer) throws {
        let resourceName = previewResourceName(for: layer)

        guard let url = Bundle.main.url(
            forResource: resourceName,
            withExtension: "wav",
            subdirectory: "previews"
        ) else {
            throw PreviewPlaybackError.missingPreview(resourceName)
        }

        do {
            player = try AVAudioPlayer(contentsOf: url)
            player?.prepareToPlay()
            player?.play()
        } catch {
            throw PreviewPlaybackError.playbackFailed(error.localizedDescription)
        }
    }

    private func previewResourceName(for layer: WaveLayer) -> String {
        switch layer.type {
        case .pulse:
            if layer.duty < 0.18 {
                return "pulse_10"
            }
            if layer.duty < 0.38 {
                return "pulse_25"
            }
            return "pulse_50"
        case .sine:
            return "sine"
        case .sawtooth:
            return "sawtooth"
        case .triangle:
            return "triangle"
        }
    }
}
