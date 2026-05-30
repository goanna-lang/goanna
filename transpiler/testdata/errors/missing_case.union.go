package main

type atom struct{}

type gender union {
	Male, Female atom
}

func main() {
	var g gender
	switch g.(union) {
	case Male:
	}
}
