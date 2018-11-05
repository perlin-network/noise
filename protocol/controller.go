package protocol

type Controller struct {
	Cancellation chan struct{}
}

func NewController() *Controller {
	return &Controller{
		Cancellation: make(chan struct{}),
	}
}
