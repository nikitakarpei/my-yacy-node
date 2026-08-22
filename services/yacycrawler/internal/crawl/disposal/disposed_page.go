package disposal

type Mark string

type DisposedPage struct {
	Mark   Mark
	Reason Reason
}
