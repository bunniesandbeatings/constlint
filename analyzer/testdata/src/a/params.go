package a

// UpdatePersonWithConstParams updates a person with const parameters.
// +const:[name, age]
func UpdatePersonWithConstParams(name string, age int, email string) {
	name = "John"              // want "assignment to const parameter"
	age = 30                   // want "assignment to const parameter"
	email = "john@example.com" // OK: not marked as const
}

// RegularFunction without const parameters
func RegularFunction(name string, age int) {
	name = "Jane" // OK: not marked as const
	age = 25      // OK: not marked as const
}

// UpdatePersonObject updates a person object but p is const.
// +const:[p]
func UpdatePersonObject(p *Person) {
	p = &Person{} // want "assignment to const parameter"

	// These are still checked by the field const checker
	p.Name = "Bob" // OK: function is a const function
	p.Age = 40     // OK: Age is not marked as const
}

// ProcessData processes data without modifying it.
// +const:[data]
func ProcessData(data []int, result *int) {
	data = append(data, 5) // want "assignment to const parameter"
	*result = data[0]      // OK: result is not marked as const
}
