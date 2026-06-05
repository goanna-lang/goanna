package main

// This tests to verify the transpiler rejects non-exhaustive switch statements

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
