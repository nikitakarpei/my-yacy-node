package documentasks

type Kind string

const (
	NamingTheDocumentsToMatch Kind = "naming the documents to match"
	OverTheCeiling            Kind = "naming none, over the ceiling"
	Skipped                   Kind = "skipped, no documents to match"
	PredictedOverTheCeiling   Kind = "naming none, predicted over the ceiling"
)
