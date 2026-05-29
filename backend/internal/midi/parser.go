package midi

import (
	"math"
	"sort"

	"gitlab.com/gomidi/midi/v2/smf"

	"octabit/backend/internal/renderer"
)

const drumChannel = 9

type ReadOptions struct {
	IncludeDrumChannel bool
}

type Profile struct {
	Channels        []int `json:"channels"`
	MelodicChannels []int `json:"melodic_channels"`
	MultiChannel    bool  `json:"multi_channel"`
}

type activeNote struct {
	startSeconds float64
	velocity     int
}

type noteKey struct {
	channel uint8
	pitch   uint8
}

func ReadNotes(path string) ([]renderer.Note, error) {
	notes, _, err := readNotesAndProfile(path, ReadOptions{})
	return notes, err
}

func ReadNotesWithOptions(path string, options ReadOptions) ([]renderer.Note, error) {
	notes, _, err := readNotesAndProfile(path, options)
	return notes, err
}

func ReadProfile(path string) (Profile, error) {
	_, profile, err := readNotesAndProfile(path, ReadOptions{})
	return profile, err
}

func readNotesAndProfile(path string, options ReadOptions) ([]renderer.Note, Profile, error) {
	midiFile, err := smf.ReadFile(path)
	if err != nil {
		return nil, Profile{}, err
	}

	activeNotes := map[noteKey][]activeNote{}
	notes := []renderer.Note{}
	channels := map[int]struct{}{}
	melodicChannels := map[int]struct{}{}
	tickToSeconds := newTickConverter(midiFile)
	for _, track := range midiFile.Tracks {
		var absTicks int64
		for _, event := range track {
			absTicks += int64(event.Delta)
			absSeconds := tickToSeconds(absTicks)
			var channel, pitch, velocity uint8
			if event.Message.GetNoteStart(&channel, &pitch, &velocity) {
				humanChannel := int(channel) + 1
				channels[humanChannel] = struct{}{}
				if channel != drumChannel {
					melodicChannels[humanChannel] = struct{}{}
				}
				if channel == drumChannel && !options.IncludeDrumChannel {
					continue
				}
				key := noteKey{channel: channel, pitch: pitch}
				activeNotes[key] = append(activeNotes[key], activeNote{
					startSeconds: absSeconds,
					velocity:     int(velocity),
				})
				continue
			}
			if event.Message.GetNoteEnd(&channel, &pitch) {
				if channel == drumChannel && !options.IncludeDrumChannel {
					continue
				}
				key := noteKey{channel: channel, pitch: pitch}
				stack := activeNotes[key]
				if len(stack) == 0 {
					continue
				}
				start := stack[0]
				if len(stack) == 1 {
					delete(activeNotes, key)
				} else {
					activeNotes[key] = stack[1:]
				}
				notes = append(notes, renderer.Note{
					Pitch:    int(pitch),
					Start:    start.startSeconds,
					End:      absSeconds,
					Velocity: start.velocity,
					Channel:  int(channel) + 1,
				})
			}
		}
	}

	sort.Slice(notes, func(left, right int) bool {
		if notes[left].Start != notes[right].Start {
			return notes[left].Start < notes[right].Start
		}
		if notes[left].Pitch != notes[right].Pitch {
			return notes[left].Pitch < notes[right].Pitch
		}
		if notes[left].End != notes[right].End {
			return notes[left].End < notes[right].End
		}
		return notes[left].Velocity < notes[right].Velocity
	})
	profile := Profile{
		Channels:        sortedChannels(channels),
		MelodicChannels: sortedChannels(melodicChannels),
	}
	profile.MultiChannel = len(profile.MelodicChannels) >= 2
	return notes, profile, nil
}

func sortedChannels(channelSet map[int]struct{}) []int {
	channels := make([]int, 0, len(channelSet))
	for channel := range channelSet {
		channels = append(channels, channel)
	}
	sort.Ints(channels)
	return channels
}

func newTickConverter(midiFile *smf.SMF) func(absTicks int64) float64 {
	resolution := metricResolution(midiFile.TimeFormat)
	if resolution <= 0 {
		return func(absTicks int64) float64 {
			return float64(midiFile.TimeAt(absTicks)) / 1_000_000.0
		}
	}

	tempoChanges := midiFile.TempoChanges()
	if len(tempoChanges) == 0 {
		return func(absTicks int64) float64 {
			return ticksToSeconds(absTicks, 0, 0, 120.0, resolution)
		}
	}

	sort.Sort(tempoChanges)
	return func(absTicks int64) float64 {
		change := tempoChanges[0]
		for _, candidate := range tempoChanges {
			if candidate.AbsTicks > absTicks {
				break
			}
			change = candidate
		}
		return ticksToSeconds(
			absTicks,
			change.AbsTicks,
			float64(change.AbsTimeMicroSec)/1_000_000.0,
			change.BPM,
			resolution,
		)
	}
}

func metricResolution(timeFormat smf.TimeFormat) float64 {
	metricTicks, ok := timeFormat.(smf.MetricTicks)
	if !ok {
		return 0
	}
	return float64(metricTicks.Resolution())
}

func ticksToSeconds(absTicks, startTicks int64, startSeconds float64, bpm float64, resolution float64) float64 {
	if bpm <= 0 || resolution <= 0 {
		return startSeconds
	}
	secondsPerTick := 60.0 / bpm / resolution
	seconds := startSeconds + float64(absTicks-startTicks)*secondsPerTick
	if seconds == 0 {
		return 0
	}
	return math.Copysign(math.Abs(seconds), seconds)
}
