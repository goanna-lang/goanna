package main

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

	switch greg.gender.(union) {
	case Male:
	case Female:
	default:
	}
}
