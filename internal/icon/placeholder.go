package icon

// Placeholder reports that the icon is still the skeleton's stand-in. Set it
// to false once the game's own icon is drawn; until then cmd/winicon refuses
// to write the exe's icon, so a release can't be built with the stand-in.
const Placeholder = false
