package x10

type DeviceMDTx07 struct {
	controller *Controller
	caption    string
	address    X10Addr
	IsOn       bool
	Volume     byte
}

func (d *DeviceMDTx07) init(controller *Controller, addr X10Addr, caption string) {
	d.controller = controller
	d.address = addr
	d.caption = caption

	controller.SendStatusRequest(addr.String())
}

func (d *DeviceMDTx07) notify(command X10Command, data uint8) {
	switch command {
	case CMD_ON:
		d.IsOn = true
	case CMD_OFF:
		d.IsOn = false
	case CMD_DIM:
		d.Volume -= uint8(uint64(data) * 64 / 210)
	case CMD_BRIGHT:
		d.Volume += uint8(uint64(data) * 64 / 210)
	}
}

func (d *DeviceMDTx07) notifyExt(command X10ExtendedCommand, data uint8) {
	switch command {
	case ECMD_PRESET_DIM:
		d.IsOn = true
		d.Volume = data
	}
}

func (d *DeviceMDTx07) Address() X10Addr {
	return d.address
}

func (d *DeviceMDTx07) Caption() string {
	return d.caption
}

func (d *DeviceMDTx07) Type() string {
	return "X10::MDTx07"
}

func (d *DeviceMDTx07) GetInfo() map[string]interface{} {
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
