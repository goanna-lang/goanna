package main

type atom struct{}

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

type deskConfig union {
	config1 normalConfig
	config2 fixedConfig
	config3 strangeConfig
}

type gender union {
	Male, Female atom
}

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

	switch greg.gender.(union) {
	case Male:
	case Female:
	default:
	}

	switch v := greg.deskConfig.(union) {
	case config1:
		_ = v.randNum
	case config2:
		_ = v.numb
	case config3:
		_ = v.randStr
	default:
	}
}
