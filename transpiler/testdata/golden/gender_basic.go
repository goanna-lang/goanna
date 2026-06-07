package main

type _genderMale struct{}
type _genderFemale struct{}

func (_genderMale) isGender()   {}
func (_genderFemale) isGender() {}

type gender interface{ isGender() }

var Male gender = _genderMale{}
var Female gender = _genderFemale{}

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
	switch greg.gender.(type) {
	case _genderMale:
	case _genderFemale:
	default:
	}

}
