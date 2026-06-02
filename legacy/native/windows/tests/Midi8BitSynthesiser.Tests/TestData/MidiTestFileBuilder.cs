using Melanchall.DryWetMidi.Common;
using Melanchall.DryWetMidi.Core;

namespace Midi8BitSynthesiser.Tests.TestData;

public static class MidiTestFileBuilder
{
    public static string CreateSingleNoteMidi(string path, int noteNumber = 60, int velocity = 100, int startTicks = 0, int lengthTicks = 480, int channel = 0)
    {
        return CreateMidi(path, [new MidiNoteSpec(noteNumber, velocity, startTicks, lengthTicks, channel)]);
    }

    public static string CreateMidi(string path, IReadOnlyList<MidiNoteSpec> notes)
    {
        var timedEvents = notes
            .SelectMany(note => new[]
            {
                new TimedMidiEvent(note.StartTicks, true, note),
                new TimedMidiEvent(note.StartTicks + note.LengthTicks, false, note),
            })
            .OrderBy(item => item.AbsoluteTime)
            .ThenBy(item => item.IsNoteOn ? 1 : 0)
            .ToList();

        var events = new List<MidiEvent>();
        var currentTime = 0L;

        foreach (var timedEvent in timedEvents)
        {
            var midiEvent = timedEvent.IsNoteOn
                ? new NoteOnEvent((SevenBitNumber)timedEvent.Note.NoteNumber, (SevenBitNumber)timedEvent.Note.Velocity)
                : new NoteOffEvent((SevenBitNumber)timedEvent.Note.NoteNumber, (SevenBitNumber)0);

            midiEvent.Channel = (FourBitNumber)timedEvent.Note.Channel;
            midiEvent.DeltaTime = timedEvent.AbsoluteTime - currentTime;
            currentTime = timedEvent.AbsoluteTime;
            events.Add(midiEvent);
        }

        events.Add(new EndOfTrackEvent());

        var midiFile = new MidiFile(new TrackChunk(events))
        {
            TimeDivision = new TicksPerQuarterNoteTimeDivision(480),
        };

        Directory.CreateDirectory(Path.GetDirectoryName(path)!);
        using var stream = File.Create(path);
        midiFile.Write(stream);

        return path;
    }

    public readonly record struct MidiNoteSpec(int NoteNumber, int Velocity, int StartTicks, int LengthTicks, int Channel);

    private readonly record struct TimedMidiEvent(long AbsoluteTime, bool IsNoteOn, MidiNoteSpec Note);
}
