package main

// Types here are just examples
// Single initials here are rubbish
type redConfig struct{ r int }
type blueConfig struct{ b int }
type greenConfig struct{ g int }

type color union {
	red   redConfig
	blue  blueConfig
	green greenConfig
}

func pick(c color) int {
	switch v := c.(union) {
	case red:
		return v.r
	case blue:
		return v.b
	case green:
		return v.g
	default:
	}
	return 0
}
