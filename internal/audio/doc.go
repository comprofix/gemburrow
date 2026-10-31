// Package audio makes all the game's sound in code. There are no audio
// files: tunes and sound effects are note tables written in Go.
//
// # Styles
//
// The preset chooses a style (config.Preset.Audio):
//
//   - Chip4: four channels, by convention 0 lead, 1 harmony, 2 bass and
//     3 drums. Each plays a pulse wave (50%, 25% or 12.5% duty), a triangle
//     or noise. Part i of a tune plays on channel i. A sound effect takes
//     over its channel while it plays; the music underneath keeps time and
//     comes back in step.
//   - Speaker: one channel, square waves only, like an old PC speaker. Only
//     the first part of a tune plays. A sound effect interrupts the music,
//     which carries on from where it stopped.
//
// # Notes, tunes and effects
//
// A Note is a pitch held for a time; Hz 0 is a rest. N("C4", 12) is a named
// note held for 12 ticks, and Notes("C4:12 E4:12 -:24") reads a list. A
// Voice gives a part its wave and volume envelope (attack, decay to a
// sustain level, release). A Tune is one Part per channel and may loop; keep
// a looping tune's parts the same length. An Effect is one Part on a
// channel with a priority: it replaces an effect already playing there only
// if its priority is at least as high. Bank.Check catches mistakes in the
// tables, and a test should call it.
//
// # Timing
//
// Game code calls Synth.Now with the tick at the start of each update. Each
// sound then starts exactly on its tick, a fixed delay (config
// AudioScheduleMs) later, however the sound device reads ahead. Without Now,
// as in tests, commands apply at once.
//
// # Volumes and tests
//
// SetVolumes sets the master, music and effects levels and SetEnabled turns
// all sound on or off; the game applies them from the settings. A nil
// *Synth ignores every call, so scenes run in tests without sound.
package audio
