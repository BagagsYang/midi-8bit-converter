import SwiftUI

@main
struct MIDI8BitSynthesiserApp: App {
    var body: some Scene {
        WindowGroup("OctaBit") {
            ContentView()
        }
        .windowResizability(.contentMinSize)
        .windowToolbarStyle(.unifiedCompact)
    }
}
