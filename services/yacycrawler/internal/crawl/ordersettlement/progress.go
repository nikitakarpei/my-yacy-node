package ordersettlement

type OrderProgress interface {
	OrderReceived()
	OrderRedelivered()
	OrderCompleted()
}
