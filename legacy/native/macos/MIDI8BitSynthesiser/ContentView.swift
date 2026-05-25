import SwiftUI
import UniformTypeIdentifiers

struct ContentView: View {
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var viewModel = SynthesiserViewModel()
    @State private var isQueueDropTargeted = false
    @State private var showsClearQueueConfirmation = false

    var body: some View {
        HStack(alignment: .top, spacing: 0) {
            queueSidebar
                .frame(width: 340)

            detailPane
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .frame(minWidth: 940, minHeight: 680)
        .confirmationDialog(
            "Clear the entire queue?",
            isPresented: $showsClearQueueConfirmation,
            titleVisibility: .visible
        ) {
            Button("Clear Queue", role: .destructive, action: viewModel.clearQueue)
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("This resets the current batch while keeping your sound design settings.")
        }
    }

    private var queueSidebar: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 8) {
                        Text("Queue")
                            .font(.title3.weight(.semibold))
                        countBadge(value: viewModel.queue.count)
                    }

                    if viewModel.queue.isEmpty {
                        Text(queueHelperText)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Spacer()

                Button(role: .destructive) {
                    showsClearQueueConfirmation = true
                } label: {
                    Image(systemName: "trash")
                }
                .buttonStyle(.borderless)
                .font(.title3)
                .help("Clear Queue")
                .accessibilityLabel("Clear Queue")
                .disabled(viewModel.queue.isEmpty)
            }

            Group {
                if viewModel.queue.isEmpty {
                    emptyQueueState
                } else {
                    List(selection: $viewModel.selectedJobIDs) {
                        ForEach(viewModel.queue) { job in
                            QueueRow(
                                job: job,
                                canRemove: !viewModel.isProcessing,
                                onRemove: {
                                    viewModel.removeJob(id: job.id)
                                }
                            )
                                .tag(job.id)
                        }
                        .onMove(perform: viewModel.moveQueue(fromOffsets:toOffset:))
                    }
                    .listStyle(.sidebar)
                    .scrollContentBackground(.hidden)
                    .background(Color.clear)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            if !viewModel.queue.isEmpty {
                HStack {
                    Spacer()
                    Button(action: viewModel.importFiles) {
                        Image(systemName: "plus")
                    }
                    .buttonStyle(.borderless)
                    .font(.title3)
                    .keyboardShortcut("o", modifiers: [.command])
                    .disabled(viewModel.isProcessing)
                }
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background {
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .fill(
                    isQueueDropTargeted
                        ? Color.accentColor.opacity(0.08)
                        : Color(nsColor: .controlBackgroundColor)
                )
        }
        .overlay {
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .strokeBorder(
                    isQueueDropTargeted ? Color.accentColor : componentBorderColor,
                    style: StrokeStyle(lineWidth: isQueueDropTargeted ? 2 : 1, dash: isQueueDropTargeted ? [10, 6] : [])
                )
        }
        .padding(16)
        .onDrop(
            of: [UTType.fileURL],
            isTargeted: $isQueueDropTargeted,
            perform: viewModel.handleDrop(providers:)
        )
    }

    private var detailPane: some View {
        VStack(alignment: .leading, spacing: 20) {
            HStack(alignment: .top, spacing: 16) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Sound Design")
                        .font(.system(size: 30, weight: .semibold, design: .rounded))
                    Text("Shape up to three waveform layers, then export the current queue as WAV.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Spacer()

                HStack(spacing: 8) {
                    summaryPill(title: "Queued", value: "\(viewModel.queue.count)")
                    summaryPill(title: "Layers", value: "\(viewModel.audibleLayerCount)")
                }
            }

            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    sectionCard {
                        HStack(alignment: .top, spacing: 12) {
                            VStack(alignment: .leading, spacing: 4) {
                                Text("Layers")
                                    .font(.title3.weight(.semibold))
                                Text("Blend waveforms with concise controls and optional pitch-dependent layer curves.")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }

                            Spacer()

                            Button(action: viewModel.addLayer) {
                                Label("Add Layer", systemImage: "plus")
                            }
                            .disabled(!viewModel.canAddLayer)
                        }

                        VStack(spacing: 14) {
                            ForEach(Array(viewModel.layers.indices), id: \.self) { index in
                                LayerEditorCard(
                                    index: index,
                                    layer: $viewModel.layers[index],
                                    canRemove: index > 0 && viewModel.canRemoveLayer,
                                    isProcessing: viewModel.isProcessing,
                                    onPreview: {
                                        viewModel.playPreview(for: viewModel.layers[index])
                                    },
                                    onRemove: {
                                        viewModel.removeLayer(at: index)
                                    }
                                )
                            }
                        }
                    }

                    sectionCard {
                        VStack(alignment: .leading, spacing: 12) {
                            Text("Output")
                                .font(.title3.weight(.semibold))

                            Text("Choose the export sample rate. Export naming matches the current web curve semantics.")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)

                            Picker("Sample Rate", selection: $viewModel.sampleRate) {
                                Text("44.1 kHz").tag(44_100)
                                Text("48 kHz").tag(48_000)
                                Text("96 kHz").tag(96_000)
                            }
                            .pickerStyle(.segmented)
                            .disabled(viewModel.isProcessing)
                        }
                    }
                }
                .frame(maxWidth: 860, alignment: .leading)
            }

            exportBar
        }
        .padding(24)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(Color(nsColor: .windowBackgroundColor))
    }

    private var emptyQueueState: some View {
        VStack(spacing: 12) {
            Image(systemName: "square.and.arrow.down")
                .font(.system(size: 30, weight: .semibold))
                .foregroundStyle(Color.accentColor)

            Text("Drop MIDI Files Here")
                .font(.headline)

            Text("Supports .mid and .midi files.")
                .font(.subheadline)
                .foregroundStyle(.secondary)

            Button("Add MIDI", action: viewModel.importFiles)
                .buttonStyle(.borderedProminent)
                .keyboardShortcut("o", modifiers: [.command])
                .disabled(viewModel.isProcessing)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(20)
        .background {
            RoundedRectangle(cornerRadius: 20, style: .continuous)
                .fill(Color.secondary.opacity(0.06))
        }
        .overlay {
            RoundedRectangle(cornerRadius: 20, style: .continuous)
                .strokeBorder(componentBorderColor, style: StrokeStyle(lineWidth: 1, dash: [8, 6]))
        }
    }

    private var exportBar: some View {
        sectionCard {
            HStack(alignment: .center, spacing: 16) {
                VStack(alignment: .leading, spacing: 8) {
                    HStack(spacing: 8) {
                        summaryPill(title: "Queue", value: "\(viewModel.queue.count)")
                        summaryPill(title: "Layers", value: "\(viewModel.layers.count)")
                        summaryPill(title: "Rate", value: sampleRateTitle(viewModel.sampleRate))
                    }

                    Text(viewModel.statusMessage)
                        .font(.subheadline.weight(.medium))

                    if let error = viewModel.lastErrorMessage {
                        statusFootnote("Error", error, tint: .red)
                    } else if let summary = viewModel.lastRunSummary {
                        statusFootnote("Summary", summary)
                    } else {
                        Text(exportHelperText)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Spacer(minLength: 16)

                if viewModel.isProcessing {
                    ProgressView()
                        .controlSize(.small)
                }

                Button(action: viewModel.startConversion) {
                    Label(viewModel.exportButtonTitle, systemImage: "waveform")
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
                .keyboardShortcut("e", modifiers: [.command])
                .disabled(!viewModel.canStartConversion)
            }
        }
    }

    private var queueHelperText: String {
        "Add MIDI files and drag rows to set the export order."
    }

    private var exportHelperText: String {
        viewModel.queue.isEmpty
            ? "Add at least one MIDI file to enable export."
            : "Export applies the current layers and sample rate to every queued file."
    }

    private func sampleRateTitle(_ value: Int) -> String {
        switch value {
        case 44_100:
            return "44.1"
        case 48_000:
            return "48"
        case 96_000:
            return "96"
        default:
            return "\(value)"
        }
    }

    private func sectionCard<Content: View>(@ViewBuilder _ content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 16, content: content)
            .padding(20)
            .background {
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .fill(Color(nsColor: .controlBackgroundColor))
            }
            .overlay {
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .strokeBorder(componentBorderColor)
            }
    }

    private var componentBorderColor: Color {
        colorScheme == .light ? Color.secondary.opacity(0.28) : Color.secondary.opacity(0.14)
    }

    private func countBadge(value: Int) -> some View {
        Text("\(value)")
            .font(.caption.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(Color.secondary.opacity(0.12), in: Capsule())
    }

    private func summaryPill(title: String, value: String) -> some View {
        HStack(spacing: 6) {
            Text(title)
                .foregroundStyle(.secondary)
            Text(value)
                .fontWeight(.semibold)
        }
        .font(.caption)
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color.secondary.opacity(0.10), in: Capsule())
    }

    private func statusFootnote(_ title: String, _ value: String, tint: Color = .accentColor) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Circle()
                .fill(tint)
                .frame(width: 8, height: 8)
                .padding(.top, 4)

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Text(value)
                    .font(.caption)
            }
        }
    }
}

private struct QueueRow: View {
    let job: ConversionJob
    let canRemove: Bool
    let onRemove: () -> Void

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Button(role: .destructive, action: onRemove) {
                Image(systemName: "minus.circle.fill")
                    .foregroundStyle(.red)
            }
            .buttonStyle(.borderless)
            .disabled(!canRemove)
            .help("Remove from Queue")

            VStack(alignment: .leading, spacing: 6) {
                Text(job.inputURL.lastPathComponent)
                    .fontWeight(.medium)
                    .lineLimit(1)

                if let statusDetail {
                    HStack(spacing: 8) {
                        Text(job.status.label)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(statusColor)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(statusColor.opacity(0.12), in: Capsule())

                        Text(statusDetail)
                            .font(.caption)
                            .foregroundStyle(statusDetailColor)
                            .lineLimit(1)
                    }
                }
            }
        }
        .padding(.vertical, 6)
    }

    private var statusColor: Color {
        switch job.status {
        case .queued:
            return .secondary
        case .processing:
            return .accentColor
        case .completed:
            return .green
        case .failed:
            return .red
        }
    }

    private var statusDetailColor: Color {
        switch job.status {
        case .failed:
            return .red
        default:
            return .secondary
        }
    }

    private var statusDetail: String? {
        switch job.status {
        case .queued:
            return nil
        case .processing:
            return "Converting now"
        case .completed(let url):
            return url.lastPathComponent
        case .failed(let message):
            return message
        }
    }
}

private struct LayerEditorCard: View {
    let index: Int
    @Binding var layer: WaveLayer
    let canRemove: Bool
    let isProcessing: Bool
    let onPreview: () -> Void
    let onRemove: () -> Void
    @State private var selectedCurvePointID: UUID?

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Label("Layer \(index + 1)", systemImage: layer.type.symbolName)
                        .font(.headline)
                    Text(layer.type.displayName)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Spacer()

                Button(action: onPreview) {
                    Label("Preview", systemImage: "play.fill")
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(isProcessing)

                if canRemove {
                    Button(role: .destructive, action: onRemove) {
                        Image(systemName: "minus.circle")
                    }
                    .buttonStyle(.borderless)
                    .help("Remove Layer")
                    .disabled(isProcessing)
                }
            }

            Picker("Waveform", selection: $layer.type) {
                ForEach(WaveformType.allCases) { waveform in
                    Text(waveform.compactLabel).tag(waveform)
                }
            }
            .pickerStyle(.segmented)
            .disabled(isProcessing)

            VStack(alignment: .leading, spacing: 12) {
                if layer.type == .pulse {
                    SliderRow(
                        title: "Pulse Width",
                        valueText: String(format: "%.2f", layer.duty),
                        value: $layer.duty,
                        range: 0.01...0.99,
                        step: 0.01,
                        isDisabled: isProcessing
                    )
                }

                SliderRow(
                    title: "Base Volume",
                    valueText: String(format: "%.1f×", layer.volume),
                    value: $layer.volume,
                    range: 0.0...2.0,
                    step: 0.1,
                    isDisabled: isProcessing
                )
            }

            Toggle("Enable frequency-gain curve", isOn: $layer.isFrequencyCurveEnabled)
                .disabled(isProcessing)
                .onChange(of: layer.isFrequencyCurveEnabled) { _, isEnabled in
                    if isEnabled {
                        layer.ensureDefaultFrequencyCurve()
                    } else {
                        selectedCurvePointID = nil
                    }
                }

            if layer.isFrequencyCurveEnabled {
                curveEditorSection
            }
        }
        .padding(18)
        .background {
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .fill(Color.secondary.opacity(0.06))
        }
        .overlay {
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .strokeBorder(Color.secondary.opacity(0.12))
        }
    }

    private var curveEditorSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text("Frequency-Gain Curve")
                    .font(.subheadline.weight(.semibold))
                Text("Layer gain is evaluated per note using a log-frequency axis and dB gain.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            FrequencyCurveEditor(
                points: $layer.frequencyCurve,
                selectedPointID: $selectedCurvePointID,
                isDisabled: isProcessing
            )
            .frame(height: 180)

            HStack {
                Text("8 Hz")
                Spacer()
                Text("440 Hz")
                Spacer()
                Text("12.5 kHz")
            }
            .font(.caption2)
            .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                Button("Add Point", action: addCurvePoint)
                    .disabled(isProcessing || layer.frequencyCurve.count >= FrequencyCurveConstants.maxPoints)

                Button("Remove Selected", action: removeSelectedCurvePoint)
                    .disabled(isProcessing || !canRemoveSelectedCurvePoint)

                Button("Reset Curve", action: resetCurve)
                    .disabled(isProcessing)
            }
            .buttonStyle(.bordered)
            .controlSize(.small)

            Text("Preview is raw waveform only for now and does not reflect this curve.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private var sortedCurve: [FrequencyCurvePoint] {
        let curve = layer.frequencyCurve.isEmpty ? FrequencyCurvePoint.defaultAnchors() : layer.frequencyCurve
        return curve.sorted { $0.frequencyHz < $1.frequencyHz }
    }

    private var canRemoveSelectedCurvePoint: Bool {
        guard let selectedCurvePointID,
              let selectedIndex = sortedCurve.firstIndex(where: { $0.id == selectedCurvePointID }) else {
            return false
        }

        return selectedIndex > 0 && selectedIndex < sortedCurve.count - 1
    }

    private func addCurvePoint() {
        layer.ensureDefaultFrequencyCurve()
        guard layer.frequencyCurve.count < FrequencyCurveConstants.maxPoints else {
            return
        }

        let sortedPoints = sortedCurve
        guard sortedPoints.count >= 2 else {
            layer.frequencyCurve = FrequencyCurvePoint.defaultAnchors()
            return
        }

        let insertionIndex = widestCurveSegmentIndex(in: sortedPoints)
        let leftPoint = sortedPoints[insertionIndex]
        let rightPoint = sortedPoints[insertionIndex + 1]
        let newPoint = FrequencyCurvePoint(
            frequencyHz: sqrt(leftPoint.frequencyHz * rightPoint.frequencyHz),
            gainDB: (leftPoint.gainDB + rightPoint.gainDB) / 2.0
        )

        layer.frequencyCurve.append(newPoint)
        layer.frequencyCurve.sort { $0.frequencyHz < $1.frequencyHz }
        selectedCurvePointID = newPoint.id
    }

    private func removeSelectedCurvePoint() {
        guard canRemoveSelectedCurvePoint,
              let selectedCurvePointID else {
            return
        }

        layer.frequencyCurve.removeAll { $0.id == selectedCurvePointID }
        self.selectedCurvePointID = nil
    }

    private func resetCurve() {
        layer.frequencyCurve = FrequencyCurvePoint.defaultAnchors()
        selectedCurvePointID = nil
    }

    private func widestCurveSegmentIndex(in points: [FrequencyCurvePoint]) -> Int {
        guard points.count >= 2 else {
            return 0
        }

        var widestIndex = 0
        var widestSpan = 0.0

        for index in 0..<(points.count - 1) {
            let span = log(points[index + 1].frequencyHz) - log(points[index].frequencyHz)
            if span > widestSpan {
                widestSpan = span
                widestIndex = index
            }
        }

        return widestIndex
    }
}

private struct SliderRow: View {
    let title: String
    let valueText: String
    @Binding var value: Double
    let range: ClosedRange<Double>
    let step: Double
    let isDisabled: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(title)
                Spacer()
                Text(valueText)
                    .font(.subheadline)
                    .monospacedDigit()
                    .foregroundStyle(.secondary)
            }

            Slider(value: $value, in: range, step: step)
                .disabled(isDisabled)
        }
    }
}

private struct FrequencyCurveEditor: View {
    @Binding var points: [FrequencyCurvePoint]
    @Binding var selectedPointID: UUID?
    let isDisabled: Bool

    var body: some View {
        GeometryReader { geometry in
            let plotRect = plotArea(in: geometry.size)
            let sortedPoints = displayedPoints

            ZStack {
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(Color.secondary.opacity(0.05))

                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .strokeBorder(Color.secondary.opacity(0.18))

                Path { path in
                    let zeroY = yPosition(forGain: 0.0, in: plotRect)
                    path.move(to: CGPoint(x: plotRect.minX, y: zeroY))
                    path.addLine(to: CGPoint(x: plotRect.maxX, y: zeroY))
                }
                .stroke(Color.secondary.opacity(0.18), style: StrokeStyle(lineWidth: 1, dash: [4, 4]))

                Path { path in
                    path.move(to: CGPoint(x: plotRect.minX, y: plotRect.midY))
                    path.addLine(to: CGPoint(x: plotRect.maxX, y: plotRect.midY))
                }
                .stroke(Color.secondary.opacity(0.12), lineWidth: 1)

                Path { path in
                    guard let firstPoint = sortedPoints.first else { return }
                    path.move(to: position(for: firstPoint, in: plotRect))
                    for point in sortedPoints.dropFirst() {
                        path.addLine(to: position(for: point, in: plotRect))
                    }
                }
                .stroke(Color.accentColor, style: StrokeStyle(lineWidth: 2.5, lineJoin: .round))

                ForEach(Array(sortedPoints.enumerated()), id: \.element.id) { index, point in
                    let pointPosition = position(for: point, in: plotRect)
                    Circle()
                        .fill(selectedPointID == point.id ? Color.accentColor : Color(nsColor: .windowBackgroundColor))
                        .overlay {
                            Circle()
                                .stroke(
                                    selectedPointID == point.id ? Color.accentColor : Color.secondary.opacity(0.65),
                                    lineWidth: selectedPointID == point.id ? 3 : 2
                                )
                        }
                        .frame(width: 14, height: 14)
                        .contentShape(Rectangle())
                        .position(pointPosition)
                        .gesture(
                            DragGesture(minimumDistance: 0)
                                .onChanged { gesture in
                                    guard !isDisabled else { return }
                                    selectedPointID = point.id
                                    updatePoint(
                                        point,
                                        sortedIndex: index,
                                        location: gesture.location,
                                        plotRect: plotRect,
                                        sortedPoints: sortedPoints
                                    )
                                }
                        )
                }
            }
            .overlay(alignment: .topLeading) {
                Text("+12 dB")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .padding(.top, 6)
                    .padding(.leading, 8)
                    .allowsHitTesting(false)
            }
            .overlay(alignment: .bottomLeading) {
                Text("-36 dB")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .padding(.bottom, 6)
                    .padding(.leading, 8)
                    .allowsHitTesting(false)
            }
        }
    }

    private var displayedPoints: [FrequencyCurvePoint] {
        let currentPoints = points.isEmpty ? FrequencyCurvePoint.defaultAnchors() : points
        return currentPoints.sorted { $0.frequencyHz < $1.frequencyHz }
    }

    private func plotArea(in size: CGSize) -> CGRect {
        CGRect(
            x: 26,
            y: 14,
            width: max(1, size.width - 40),
            height: max(1, size.height - 28)
        )
    }

    private func position(for point: FrequencyCurvePoint, in plotRect: CGRect) -> CGPoint {
        CGPoint(
            x: xPosition(forFrequency: point.frequencyHz, in: plotRect),
            y: yPosition(forGain: point.gainDB, in: plotRect)
        )
    }

    private func xPosition(forFrequency frequencyHz: Double, in plotRect: CGRect) -> CGFloat {
        let minLog = log(FrequencyCurveConstants.minFrequencyHz)
        let maxLog = log(FrequencyCurveConstants.maxFrequencyHz)
        let ratio = (log(frequencyHz) - minLog) / (maxLog - minLog)
        return plotRect.minX + (plotRect.width * ratio)
    }

    private func yPosition(forGain gainDB: Double, in plotRect: CGRect) -> CGFloat {
        let ratio = (FrequencyCurveConstants.maxGainDB - gainDB)
            / (FrequencyCurveConstants.maxGainDB - FrequencyCurveConstants.minGainDB)
        return plotRect.minY + (plotRect.height * ratio)
    }

    private func frequency(at x: CGFloat, in plotRect: CGRect) -> Double {
        let clampedRatio = max(0.0, min(1.0, (x - plotRect.minX) / plotRect.width))
        let scale = FrequencyCurveConstants.maxFrequencyHz / FrequencyCurveConstants.minFrequencyHz
        return FrequencyCurveConstants.minFrequencyHz * pow(scale, clampedRatio)
    }

    private func gain(at y: CGFloat, in plotRect: CGRect) -> Double {
        let clampedRatio = max(0.0, min(1.0, (y - plotRect.minY) / plotRect.height))
        return FrequencyCurveConstants.maxGainDB
            - (clampedRatio * (FrequencyCurveConstants.maxGainDB - FrequencyCurveConstants.minGainDB))
    }

    private func updatePoint(
        _ point: FrequencyCurvePoint,
        sortedIndex: Int,
        location: CGPoint,
        plotRect: CGRect,
        sortedPoints: [FrequencyCurvePoint]
    ) {
        guard let pointIndex = points.firstIndex(where: { $0.id == point.id }) else {
            return
        }

        let clampedY = max(plotRect.minY, min(plotRect.maxY, location.y))
        var updatedPoint = points[pointIndex]
        updatedPoint.gainDB = gain(at: clampedY, in: plotRect)

        if sortedIndex == 0 {
            updatedPoint.frequencyHz = FrequencyCurveConstants.minFrequencyHz
        } else if sortedIndex == sortedPoints.count - 1 {
            updatedPoint.frequencyHz = FrequencyCurveConstants.maxFrequencyHz
        } else {
            let minimumX = xPosition(
                forFrequency: sortedPoints[sortedIndex - 1].frequencyHz,
                in: plotRect
            ) + 12
            let maximumX = xPosition(
                forFrequency: sortedPoints[sortedIndex + 1].frequencyHz,
                in: plotRect
            ) - 12
            let clampedX = max(minimumX, min(maximumX, location.x))
            updatedPoint.frequencyHz = frequency(at: clampedX, in: plotRect)
        }

        points[pointIndex] = updatedPoint
        points.sort { $0.frequencyHz < $1.frequencyHz }
    }
}
