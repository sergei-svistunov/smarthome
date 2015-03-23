package x10

type Device interface {
	init(controller *Controller, addr byte, caption string)
	Address() byte
	Caption() string
	Type() string
	GetInfo() map[string]interface{}
}
