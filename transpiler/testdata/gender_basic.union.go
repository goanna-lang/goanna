package main

// TODO: This should be defined by the transpiler as a global type. 
type atom struct{}

type gender union {
	Male, Female atom
}

type person struct {
	name   string
	gender gender
}

func main() {
	greg := person{
		name:   "Greg",
		gender: Male,
	}

	// For testing purposes. Each case handled even though nothing is run
	switch greg.gender.(union) {
	case Male:
	case Female:
	default:
	}
}
