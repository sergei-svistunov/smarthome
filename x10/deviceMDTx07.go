package x10

type DeviceMDTx07 struct {
	controller *Controller
	caption    string
	address    byte
	IsOn       bool
	Volume     byte
}

func (d* DeviceMDTx07) init(controller *Controller, addr byte, caption string) {
	d.controller = controller
	d.address = addr
	d.caption = caption
}

func (d* DeviceMDTx07) Address() byte {
	return d.address
}

func (d* DeviceMDTx07) Caption() string {
	return d.caption
}

func (d* DeviceMDTx07) Type() string {
	return "X10::MDTx07"
}

func (d* DeviceMDTx07) GetInfo() map[string]interface{} {
	res := map[string]interface{}{
		"is_on":  d.IsOn,
		"volume": d.Volume,
	}

	return res
}

func NewDeviceMDTx07() *DeviceMDTx07 {
	d := &DeviceMDTx07{}

	return d
}
