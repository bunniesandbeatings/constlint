package a

// Person represents a person with a mix of immutable/mutable properties.
type Person struct {
	// Name marked as a const prior to the field
	// +const
	Name string

	// +const
	// PreferredName marked as a const prior to the field and doc comment
	PreferredName string

	// This is mutable
	Age int

	// Email is marked as a const inline
	Email string // +const
}

// NewPerson creates a new person.
func NewPerson(name, preferred, email string, age int) *Person {
	return &Person{
		Name:          name,      // OK: in constructor
		PreferredName: preferred, // OK: in constructor
		Age:           age,       // OK: not marked const
		Email:         email,     // OK: in constructor
	}
}

// SetName sets the name of the person.
func (p *Person) SetName(name string) {
	p.Name = name // want "assignment to const field"
}

// SetPreferredName sets the preferred name of the person.
func (p *Person) SetPreferredName(name string) {
	p.PreferredName = name // want "assignment to const field"
}

// SetAge sets the age of the person.
func (p *Person) SetAge(age int) {
	p.Age = age // OK: Age is not marked as const
}

// SetEmail sets the email of the person.
func (p *Person) SetEmail(email string) {
	p.Email = email // want "assignment to const field"
}

// UpdatePerson updates a person's fields.
func UpdatePerson(p *Person) {
	p.Name = "John"              // want "assignment to const field"
	p.PreferredName = "John"     // want "assignment to const field"
	p.Age = 30                   // OK: Age is not marked as const
	p.Email = "john@example.com" // want "assignment to const field"
}

// CreatePerson is a constructor function.
func CreatePerson() *Person {
	p := &Person{}
	p.Name = "Alice"              // OK: in constructor
	p.PreferredName = "Alice"     // OK: in constructor
	p.Age = 25                    // OK: Age is not marked as const
	p.Email = "alice@example.com" // OK: in constructor
	return p
}
