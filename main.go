package main

func main() {

	// standard enum
	type role string

	const (
		RoleManager  role = "manager"
		RoleEmployee role = "employee"
	)

	// ioto enum
	type store int

	const (
		Store1 store = iota
		Store2
		Store3
	)

	//proposed syntax of sum types as non-interface union type
	// see https://github.com/golang/go/issues/76920

	// similar to how an any is interface{}
	type atom struct{}

	type normalConfig struct {
			people int
			randNum int
		}

	type fixedConfig struct {
			people int
			numb int = 10
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
		name  string
		age   int
		role  role
		store store
		gender gender
		deskConfig deskConfig
	}

  // construct a struct like normal

  greg := person{
	name: "Greg",
	age: 17,
	role: RoleManager,
	store: Store2,
	gender: Male,
	deskConfig: normalConfig{
		people: 10,
		randNum: 3,
	}
}

	// exhaustive union switch — compiler error if any case missing
	switch greg.gender.(union) {
	case Male:
	case Female:
	default: // zero value (atom / unset)
	}

	// binding form — v takes type of matched variant's payload
	switch v := greg.deskConfig.(union) {
	case config1:
		// v has type normalConfig
		_ = v.randNum
	case config2:
		// v has type fixedConfig
		_ = v.numb
	case config3:
		// v has type strangeConfig
		_ = v.randStr
	default: // zero value (atom / unset)
	}

	// default makes switch non-exhaustive — no compiler error
	switch v := greg.deskConfig.(union) {
	case config1:
		_ = v.people
	default:
	}

	// compiler rejects this — missing config2, config3
	// switch greg.deskConfig.(union) {
	// case config1:
	// }
	// Error: Failed to handle all cases.

}
