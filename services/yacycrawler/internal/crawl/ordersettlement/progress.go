package ordersettlement

type Progress interface {
	OrderReceived()
	OrderRedelivered()
	OrderCompleted()
}
