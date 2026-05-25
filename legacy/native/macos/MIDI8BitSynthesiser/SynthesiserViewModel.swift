import AppKit
import Foundation
import UniformTypeIdentifiers

@MainActor
final class SynthesiserViewModel: ObservableObject {
    @Published var queue: [ConversionJob] = []
    @Published var selectedJobIDs: Set<UUID> = []
    @Published var layers: [WaveLayer] = [
        WaveLayer(type: .pulse, duty: 0.5, volume: 1.0)
    ]
    @Published var sampleRate: Int = 48_000
    @Published var isProcessing = false
    @Published var statusMessage = "Add MIDI files, choose your layer blend, then export a WAV batch."
    @Published var lastRunSummary: String?
    @Published var lastErrorMessage: String?

    let previewPlayer = AudioPreviewPlayer()

    private let supportedExtensions = Set(["mid", "midi"])

    var canAddLayer: Bool { layers.count < 3 && !isProcessing }
    var canRemoveLayer: Bool { layers.count > 1 && !isProcessing }
    var canClearQueue: Bool { !queue.isEmpty && !isProcessing }
    var canStartConversion: Bool { !queue.isEmpty && !isProcessing }
    var exportButtonTitle: String { isProcessing ? "Processing…" : "Export WAV" }
    var selectedJobCount: Int { selectedJobIDs.count }

    func importFiles() {
        guard !isProcessing else { return }

        let panel = NSOpenPanel()
        panel.allowedContentTypes = midiContentTypes
        panel.allowsMultipleSelection = true
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.message = "Choose one or more MIDI files to add to the queue."

        guard panel.runModal() == .OK else { return }
        addFiles(urls: panel.urls)
    }

    func handleDrop(providers: [NSItemProvider]) -> Bool {
        guard !isProcessing else { return false }

        let matchingProviders = providers.filter {
            $0.hasItemConformingToTypeIdentifier(UTType.fileURL.identifier)
        }

        guard !matchingProviders.isEmpty else { return false }

        let lock = NSLock()
        let group = DispatchGroup()
        var droppedURLs: [URL] = []

        for provider in matchingProviders {
            group.enter()
            provider.loadItem(forTypeIdentifier: UTType.fileURL.identifier, options: nil) {
                item, _ in
                defer { group.leave() }

                if let data = item as? Data,
                   let url = URL(dataRepresentation: data, relativeTo: nil) {
                    lock.lock()
                    droppedURLs.append(url)
                    lock.unlock()
                } else if let url = item as? URL {
                    lock.lock()
                    droppedURLs.append(url)
                    lock.unlock()
                }
            }
        }

        group.notify(queue: .main) { [weak self] in
            guard let self else { return }
            Task { @MainActor in
                self.addFiles(urls: droppedURLs)
            }
        }

        return true
    }

    func addLayer() {
        guard canAddLayer else { return }

        let defaults: [WaveLayer] = [
            WaveLayer(type: .sine, duty: 0.5, volume: 0.5),
            WaveLayer(type: .triangle, duty: 0.5, volume: 0.5),
        ]

        let nextDefault = defaults[min(layers.count - 1, defaults.count - 1)]
        layers.append(nextDefault)
    }

    func removeLayer() {
        guard canRemoveLayer else { return }
        layers.removeLast()
    }

    func removeLayer(at index: Int) {
        guard canRemoveLayer, layers.indices.contains(index), index > 0 else { return }
        layers.remove(at: index)
    }

    func moveQueue(fromOffsets: IndexSet, toOffset: Int) {
        guard !isProcessing else { return }
        queue.move(fromOffsets: fromOffsets, toOffset: toOffset)
    }

    func removeJob(id: UUID) {
        guard !isProcessing else { return }
        queue.removeAll { $0.id == id }
        selectedJobIDs.remove(id)
    }

    func clearQueue() {
        guard !queue.isEmpty else { return }
        queue.removeAll()
        selectedJobIDs.removeAll()
        lastRunSummary = nil
        lastErrorMessage = nil
        statusMessage = "Queue cleared. Add MIDI files to start another batch."
    }

    func playPreview(for layer: WaveLayer) {
        do {
            try previewPlayer.playPreview(for: layer)
            lastErrorMessage = nil
        } catch {
            lastErrorMessage = error.localizedDescription
        }
    }

    func startConversion() {
        guard !queue.isEmpty, !isProcessing else { return }

        let outputDirectory = chooseOutputDirectory(
            defaultDirectory: queue.first?.inputURL.deletingLastPathComponent()
        )

        guard let outputDirectory else {
            statusMessage = "Export cancelled."
            return
        }

        let preparedLayers = WaveLayerExportSanitizer.sanitizedLayers(from: layers)
        let preparedRate = sampleRate

        for index in queue.indices {
            queue[index].outputDirectoryURL = outputDirectory
            queue[index].sampleRate = preparedRate
            queue[index].layers = preparedLayers
            queue[index].status = .queued
        }

        isProcessing = true
        lastRunSummary = nil
        lastErrorMessage = nil

        Task {
            await runBatch(
                outputDirectory: outputDirectory,
                sampleRate: preparedRate,
                layers: preparedLayers
            )
        }
    }

    private func addFiles(urls: [URL]) {
        guard !isProcessing else { return }

        let midiFiles = urls
            .filter { supportedExtensions.contains($0.pathExtension.lowercased()) }

        guard !midiFiles.isEmpty else {
            lastErrorMessage = "Only .mid and .midi files can be queued."
            return
        }

        lastErrorMessage = nil
        queue.append(contentsOf: midiFiles.map { ConversionJob(inputURL: $0) })
        statusMessage = "\(queue.count) file(s) queued for export."
    }

    private func chooseOutputDirectory(defaultDirectory: URL?) -> URL? {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false
        panel.directoryURL = defaultDirectory
        panel.prompt = "Choose Export Folder"
        panel.message = "Choose a folder for the generated WAV files."

        return panel.runModal() == .OK ? panel.url : nil
    }

    private func runBatch(
        outputDirectory: URL,
        sampleRate: Int,
        layers: [WaveLayer]
    ) async {
        let jobIDs = queue.map(\.id)
        var completedCount = 0
        var failedCount = 0

        for (offset, jobID) in jobIDs.enumerated() {
            guard let index = queue.firstIndex(where: { $0.id == jobID }) else { continue }

            let inputURL = queue[index].inputURL
            let outputURL = OutputFileNameBuilder.outputURL(
                for: inputURL,
                in: outputDirectory,
                layers: layers
            )

            queue[index].status = .processing
            statusMessage = "Processing \(offset + 1) of \(jobIDs.count): \(inputURL.lastPathComponent)"

            do {
                try await PythonBridge.convert(
                    inputURL: inputURL,
                    outputURL: outputURL,
                    sampleRate: sampleRate,
                    layers: layers
                )
                queue[index].status = .completed(outputURL)
                completedCount += 1
            } catch {
                queue[index].status = .failed(error.localizedDescription)
                failedCount += 1
                lastErrorMessage = error.localizedDescription
            }
        }

        isProcessing = false
        statusMessage = "Finished \(jobIDs.count) file(s)."
        lastRunSummary = "\(completedCount) completed, \(failedCount) failed."
    }

    var audibleLayerCount: Int {
        layers.filter { $0.volume > 0 }.count
    }

    private var midiContentTypes: [UTType] {
        [UTType(filenameExtension: "mid"), UTType(filenameExtension: "midi")].compactMap { $0 }
    }
}
