// enum
package stp

type PortInfoState int

const (
	PortInfoStateMine PortInfoState = iota
	PortInfoStateAged
	PortInfoStateReceived
	PortInfoStateDisabled
)

type PortDesignatedRcvInfo int

const (
	SuperiorDesignatedInfo PortDesignatedRcvInfo = iota
	RepeatedDesignatedInfo
	InferiorDesignatedInfo
	InferiorRootAlternateInfo
)

type PortRole int

const (
	PortRoleBridgePort PortRole = iota + 1
	PortRoleRootPort
	PortRoleDesignatedPort
	PortRoleAlternatePort
	PortRoleBackupPort
	PortRoleDisabledPort
)

type PointToPointMac int

const (
	StpPointToPointForceTrue  = 0
	StpPointToPointForceFalse = 1
	StpPointToPointAuto       = 2
)
