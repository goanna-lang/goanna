package main

// Types here are just examples
// Single initials here are rubbish
type redConfig struct{ r int }
type blueConfig struct{ b int }
type greenConfig struct{ g int }

func (redConfig) isColor()   {}
func (blueConfig) isColor()  {}
func (greenConfig) isColor() {}

type color interface{ isColor() }

func pick(c color) int {
	switch v := c.(type) {
	case redConfig:
		return v.r
	case blueConfig:
		return v.b
	case greenConfig:
		return v.g
	default:
	}

	return 0
}
