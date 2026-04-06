import SwiftUI

@main
struct MIDI8BitSynthesiserApp: App {
    var body: some Scene {
        WindowGroup("MIDI-8bit Synthesiser") {
            ContentView()
        }
        .windowResizability(.contentMinSize)
        .windowToolbarStyle(.unifiedCompact)
    }
}
