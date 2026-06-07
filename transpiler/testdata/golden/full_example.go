package main

type normalConfig struct {
	people  int
	randNum int
}

type fixedConfig struct {
	people int
	numb   int
}

type strangeConfig struct {
	randStr string
}

func (normalConfig) isDeskConfig()  {}
func (fixedConfig) isDeskConfig()   {}
func (strangeConfig) isDeskConfig() {}

type deskConfig interface{ isDeskConfig() }

type _genderMale struct{}
type _genderFemale struct{}

func (_genderMale) isGender()   {}
func (_genderFemale) isGender() {}

type gender interface{ isGender() }

var Male gender = _genderMale{}
var Female gender = _genderFemale{}

type person struct {
	name       string
	age        int
	gender     gender
	deskConfig deskConfig
}

func main() {
	greg := person{
		name:   "Greg",
		age:    17,
		gender: Male,
		deskConfig: normalConfig{
			people:  10,
			randNum: 3,
		},
	}

	switch greg.gender.(type) {
	case _genderMale:
	case _genderFemale:
	default:
	}

	switch v := greg.deskConfig.(type) {
	case normalConfig:
		_ = v.randNum
	case fixedConfig:
		_ = v.numb
	case strangeConfig:
		_ = v.randStr
	default:
	}

}
