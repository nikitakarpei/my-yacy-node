package pagerecall

type RepresentationKind string

type Representation any

type RecalledRepresentation struct {
	Kind           RepresentationKind
	Representation Representation
}

type RecalledPage struct {
	Representations  []RecalledRepresentation
	UnavailableKinds []RepresentationKind
}
