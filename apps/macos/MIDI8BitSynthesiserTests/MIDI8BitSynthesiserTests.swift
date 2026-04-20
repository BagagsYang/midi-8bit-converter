import XCTest
@testable import MIDI8BitSynthesiser

final class MIDI8BitSynthesiserTests: XCTestCase {
    func testWaveLayerDefaultsToDisabledCurveWithAnchorPointsSeeded() {
        let layer = WaveLayer(type: .pulse)

        XCTAssertFalse(layer.isFrequencyCurveEnabled)
        XCTAssertEqual(2, layer.frequencyCurve.count)
        XCTAssertEqual(FrequencyCurveConstants.minFrequencyHz, layer.frequencyCurve[0].frequencyHz)
        XCTAssertEqual(FrequencyCurveConstants.maxFrequencyHz, layer.frequencyCurve[1].frequencyHz)
        XCTAssertEqual([], layer.exportedFrequencyCurve)
    }

    func testDisabledCurveExportsEmptyFrequencyCurve() {
        let layer = WaveLayer(
            type: .sine,
            duty: 0.5,
            volume: 1.0,
            isFrequencyCurveEnabled: false,
            frequencyCurve: [
                FrequencyCurvePoint(frequencyHz: 440.0, gainDB: -6.0)
            ]
        )

        let payloads = LayerPayloadEncoder.runtimePayloads(from: [layer])
        XCTAssertEqual([], payloads[0].frequencyCurve)
    }

    func testEnabledCurveExportsSortedPoints() {
        let layer = WaveLayer(
            type: .triangle,
            duty: 0.5,
            volume: 0.7,
            isFrequencyCurveEnabled: true,
            frequencyCurve: [
                FrequencyCurvePoint(frequencyHz: 880.0, gainDB: 3.0),
                FrequencyCurvePoint(frequencyHz: 220.0, gainDB: -12.0),
            ]
        )

        let payloads = LayerPayloadEncoder.runtimePayloads(from: [layer])
        XCTAssertEqual(
            [
                RuntimeFrequencyCurvePointPayload(frequencyHz: 220.0, gainDB: -12.0),
                RuntimeFrequencyCurvePointPayload(frequencyHz: 880.0, gainDB: 3.0),
            ],
            payloads[0].frequencyCurve
        )
    }

    func testSanitizerFallsBackToPulseWhenAllLayersAreSilent() {
        let layers = WaveLayerExportSanitizer.sanitizedLayers(from: [
            WaveLayer(type: .sine, volume: 0.0),
            WaveLayer(type: .triangle, volume: 0.0),
        ])

        XCTAssertEqual(1, layers.count)
        XCTAssertEqual(.pulse, layers[0].type)
        XCTAssertEqual(0.5, layers[0].duty)
        XCTAssertEqual(1.0, layers[0].volume)
        XCTAssertFalse(layers[0].isFrequencyCurveEnabled)
    }

    func testOutputSuffixUsesWaveTypeForSingleLayerWithoutCurve() {
        let suffix = OutputFileNameBuilder.outputSuffix(for: [
            WaveLayer(type: .pulse, duty: 0.5, volume: 1.0)
        ])

        XCTAssertEqual("pulse", suffix)
    }

    func testOutputSuffixUsesMixForMultipleLayersWithoutCurve() {
        let suffix = OutputFileNameBuilder.outputSuffix(for: [
            WaveLayer(type: .pulse, duty: 0.5, volume: 1.0),
            WaveLayer(type: .triangle, duty: 0.5, volume: 0.5),
        ])

        XCTAssertEqual("mix", suffix)
    }

    func testCurveBearingOutputUsesExpectedHashSuffix() {
        let layer = WaveLayer(
            type: .pulse,
            duty: 0.5,
            volume: 1.0,
            isFrequencyCurveEnabled: true,
            frequencyCurve: FrequencyCurvePoint.defaultAnchors()
        )

        XCTAssertEqual(
            "[{\"duty\":0.5,\"frequency_curve\":[{\"frequency_hz\":8.175798915643707,\"gain_db\":0.0},{\"frequency_hz\":12543.853951415975,\"gain_db\":0.0}],\"type\":\"pulse\",\"volume\":1.0}]",
            LayerPayloadEncoder.jsonString(for: [layer])
        )
        XCTAssertEqual("pulse_dee027b8", OutputFileNameBuilder.outputSuffix(for: [layer]))
    }
}
