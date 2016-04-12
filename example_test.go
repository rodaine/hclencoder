package hclencoder

import (
	"log"

	"fmt"
)

func Example() {
	type Farm struct {
		Name     string    `hcl:"name"`
		Owned    bool      `hcl:"owned"`
		Location []float64 `hcl:"location"`
	}

	type Farmer struct {
		Name                 string `hcl:"name"`
		Age                  int    `hcl:"age"`
		SocialSecurityNumber string `hcle:"omit"`
	}

	type Animal struct {
		Name  string `hcl:",key"`
		Sound string `hcl:"says" hcle:"omitempty"`
	}

	type Config struct {
		Farm      `hcl:",squash"`
		Farmer    Farmer            `hcl:"farmer"`
		Animals   []Animal          `hcl:"animal"`
		Buildings map[string]string `hcl:"buildings"`
	}

	input := Config{
		Farm: Farm{
			Name:     "Ol' McDonald's Farm",
			Owned:    true,
			Location: []float64{12.34, -5.67},
		},
		Farmer: Farmer{
			Name:                 "Robert Beauregard-Michele McDonald, III",
			Age:                  65,
			SocialSecurityNumber: "please-dont-share-me",
		},
		Animals: []Animal{
			{
				Name:  "cow",
				Sound: "moo",
			},
			{
				Name:  "pig",
				Sound: "oink",
			},
			{
				Name: "rock",
			},
		},
		Buildings: map[string]string{
			"House": "123 Numbers Lane",
			"Barn":  "456 Digits Drive",
		},
	}

	hcl, err := Encode(input)
	if err != nil {
		log.Fatal("unable to encode: ", err)
	}

	fmt.Print(string(hcl))

	// Output:
	// name = "Ol' McDonald's Farm"
	//
	// owned = true
	//
	// location = [
	//   12.34,
	//   -5.67,
	// ]
	//
	// farmer {
	//   name = "Robert Beauregard-Michele McDonald, III"
	//   age  = 65
	// }
	//
	// animal "cow" {
	//   says = "moo"
	// }
	//
	// animal "pig" {
	//   says = "oink"
	// }
	//
	// animal "rock" {}
	//
	// buildings {
	//   Barn  = "456 Digits Drive"
	//   House = "123 Numbers Lane"
	// }
	//
}
