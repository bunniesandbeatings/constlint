package a

type OtherObject struct {
	P *Person
}

func NewOtherObject() *OtherObject {
	return &OtherObject{}
}

// Ensure a method is allowed as a "constructor"
func (o *OtherObject) SetPersonDetails(name, preferred, email string, age int) {
	o.P = &Person{}
	o.P.Name = name
	o.P.PreferredName = preferred
	o.P.Email = email
	o.P.Age = age
}
