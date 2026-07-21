package pagerecall

type Kind string

type Representation any

type RecalledRepresentation struct {
	Kind    Kind
	Content Representation
}

type Result struct {
	Representations []RecalledRepresentation
	Unavailable     []Kind
}
