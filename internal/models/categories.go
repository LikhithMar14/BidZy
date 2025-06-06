package models

type Categories int

// iota means that the first value is 0, the second is 1, and so on.
//idiomatic way of using iota is to use it in a const block.
const (
	ART Categories = iota
	COLLECTABLES 
	ELECTRONICS
	VEHICLES
	WATCHES
	FASHION
	SHOES
	REALESTATE
	FURNITURE
	MISCELLANEOUS
)

func (c Categories) String() string {
	names := []string{
		"ART",
		"COLLECTABLES",
		"ELECTRONICS",
		"VEHICLES",
		"WATCHES",
		"FASHION",
		"SHOES",
		"REALESTATE",
		"FURNITURE",
		"MISCELLANEOUS",
	}

	if int(c) < 0 || int(c) >= len(names) {
		return "UNKNOWN"
	}
	return names[c]
}
// usage will be like this

// var cat Categories = WATCHES // cat = 4

// fmt.Println(cat)        // prints: 4 (the integer value)
// fmt.Println(cat.String()) // prints: WATCHES (the string name)