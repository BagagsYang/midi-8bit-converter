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
                                Text("Blend waveforms with concise controls and inline previews.")
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

                            Text("Choose the export sample rate. Output naming stays unchanged.")
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
                    title: "Volume",
                    valueText: String(format: "%.1f×", layer.volume),
                    value: $layer.volume,
                    range: 0.0...2.0,
                    step: 0.1,
                    isDisabled: isProcessing
                )
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
