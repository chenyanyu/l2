// config
package lacp

import (
	"fmt"
	//"sync"
	"net"
	"time"
)

const (
	LaAggTypeLACP = iota + 1
	LaAggTypeSTATIC
)

const PortConfigModuleStr = "Port Config"

type LaAggregatorConfig struct {
	// GET-SET
	AggName string
	// GET-SET
	AggActorSystemID [6]uint8
	// GET-SET
	AggActorSystemPriority uint16
	// GET-SET
	AggActorAdminKey uint16
	// GET-SET   up/down enum
	AggAdminState bool
	// GET-SET  enable/disable enum
	AggLinkUpDownNotificationEnable bool
	// GET-SET 10s of microseconds
	AggCollectorMaxDelay uint16
	// GET-SET
	AggPortAlgorithm [3]uint8
	// GET-SET
	AggPartnerAdminPortAlgorithm [3]uint8
	// GET-SET up to 4096 values conversationids
	AggConversationAdminLink []int
	// GET-SET
	AggPartnerAdminPortConverstaionListDigest [16]uint8
	// GET-SET
	AggAdminDiscardWrongConversation bool
	// GET-SET 4096 values
	AggAdminServiceConversationMap []int
	// GET-SET
	AggPartnerAdminConvServiceMappingDigest [16]uint8
}

type LacpConfigInfo struct {
	Interval time.Duration
	Mode     uint32
	// In format AA:BB:CC:DD:EE:FF
	SystemIdMac    string
	SystemPriority uint16
}

type LaAggConfig struct {
	// Aggregator name
	Name string
	// Aggregator_MAC_address
	Mac [6]uint8
	// Aggregator_Identifier
	Id int
	// Actor_Admin_Aggregator_Key
	Key uint16
	// Aggregator Type, LACP or STATIC
	Type uint32
	// Minimum number of links
	MinLinks uint16
	// Enabled
	Enabled bool
	// LAG_ports
	LagMembers []uint16

	// System to attach this agg to
	Lacp LacpConfigInfo

	// mau properties of each link
	Properties PortProperties

	// TODO hash config
}

type LaAggPortConfig struct {

	// Actor_Port_Number
	Id uint16
	// Actor_Port_priority
	Prio uint16
	// Actor Admin Key
	Key uint16
	// Actor Oper Key
	//OperKey uint16
	// Actor_Port_Aggregator_Identifier
	AggId int

	// Admin Enable/Disable
	Enable bool

	// lacp mode On/Active/Passive
	Mode int

	// lacp timeout SHORT/LONG
	Timeout time.Duration

	// Port capabilities and attributes
	Properties PortProperties

	// System to attach this agg to
	SysId net.HardwareAddr

	// Linux If
	TraceEna bool
	IntfId   string
}

func SaveLaAggConfig(ac *LaAggConfig) {
	var a *LaAggregator
	if LaFindAggByName(ac.Name, &a) {
		netMac, _ := net.ParseMAC(ac.Lacp.SystemIdMac)
		sysId := LacpSystem{
			actor_System:          convertNetHwAddressToSysIdKey(netMac),
			Actor_System_priority: ac.Lacp.SystemPriority,
		}
		a.AggName = ac.Name
		a.aggId = ac.Id
		a.aggMacAddr = sysId.actor_System
		a.actorAdminKey = ac.Key
		a.AggType = ac.Type
		a.AggMinLinks = ac.MinLinks
		a.Config = ac.Lacp
	}
}

func CreateLaAgg(agg *LaAggConfig) {

	//var wg sync.WaitGroup

	a := NewLaAggregator(agg)
	fmt.Printf("%#v\n", a)

	/*
		// two methods for creating ports after CreateLaAgg is created
		// 1) PortNumList is populated
		// 2) find Key's that match
		for _, pId := range a.PortNumList {
			wg.Add(1)
			go func(pId uint16) {
				var p *LaAggPort
				defer wg.Done()
				if LaFindPortById(pId, &p) && p.aggSelected == LacpAggUnSelected {
					// if aggregation has been provided then lets kick off the process
					p.checkConfigForSelection()
				}
			}(pId)
		}

		wg.Wait()
	*/
	index := 0
	var p *LaAggPort
	fmt.Println("looking for ports with actorAdminKey ", a.actorAdminKey)
	if mac, err := net.ParseMAC(a.Config.SystemIdMac); err == nil {
		if sgi := LacpSysGlobalInfoByIdGet(LacpSystem{actor_System: convertNetHwAddressToSysIdKey(mac),
			Actor_System_priority: a.Config.SystemPriority}); sgi != nil {
			for index != -1 {
				fmt.Println("looking for ", index)
				if LaFindPortByKey(a.actorAdminKey, &index, &p) {
					if p.aggSelected == LacpAggUnSelected {
						AddLaAggPortToAgg(a.aggId, p.PortNum)
					}
				} else {
					break
				}
			}
		}
	}
}

func DeleteLaAgg(Id int) {
	var a *LaAggregator
	if LaFindAggById(Id, &a) {

		for _, pId := range a.PortNumList {
			DeleteLaAggPortFromAgg(Id, pId)
			DeleteLaAggPort(pId)
		}

		a.aggId = 0
		a.actorAdminKey = 0
		a.partnerSystemId = [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		a.ready = false
	}
}

func DisableLaAgg(Id int) {
	var a *LaAggregator
	if LaFindAggById(Id, &a) {

		for _, pId := range a.PortNumList {
			DisableLaAggPort(pId)
		}
	}
}

func CreateLaAggPort(port *LaAggPortConfig) {
	var pTmp *LaAggPort

	// sanity check that port does not exist already
	if !LaFindPortById(port.Id, &pTmp) {
		p := NewLaAggPort(port)

		fmt.Println("Port mode", port.Mode)
		// Is lacp enabled or not
		if port.Mode != LacpModeOn {
			p.lacpEnabled = true
			// make the port aggregatable
			LacpStateSet(&p.actorAdmin.State, LacpStateAggregationBit)
			// set the activity State
			if port.Mode == LacpModeActive {
				LacpStateSet(&p.actorAdmin.State, LacpStateActivityBit)
			} else {
				LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
			}
		} else {
			// port is not aggregatible
			LacpStateClear(&p.actorAdmin.State, LacpStateAggregationBit)
			LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
			p.lacpEnabled = false
		}

		if port.Timeout == LacpShortTimeoutTime {
			LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.actorAdmin.State, LacpStateTimeoutBit)
		}

		// lets start all the State machines
		p.BEGIN(false)
		p.LaPortLog(fmt.Sprintf("Creating LaAggPort %d", port.Id))

		// TODO: need logic to check link status
		p.LinkOperStatus = true

		if p.LinkOperStatus && port.Enable {

			if p.Key != 0 {
				var a *LaAggregator
				if LaFindAggByKey(p.Key, &a) {
					p.LaPortLog("Found Agg by Key, attaching port to agg")
					// If the agg is defined lets add port to
					AddLaAggPortToAgg(a.aggId, p.PortNum)
				}
			}

			// if port is enabled and lacp is enabled
			p.LaAggPortEnabled()

			// check for selection
			p.checkConfigForSelection()

		}
		fmt.Printf("PORT (after config create):\n%#v\n", p)
	} else {
		fmt.Println("CONF: ERROR PORT ALREADY EXISTS")
	}
}

func DeleteLaAggPort(pId uint16) {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		if LaAggPortNumListPortIdExist(p.AggId, pId) {
			fmt.Println("CONF: ERROR Must detach p", pId, "from agg", p.AggId, "before deletion")
			return
		}
		if p.PortEnabled {
			DisableLaAggPort(p.PortNum)
		}

		p.DelLaAggPort()
	}
}

func DisableLaAggPort(pId uint16) {
	var p *LaAggPort

	// port exists
	// port exists in agg exists
	if LaFindPortById(pId, &p) &&
		LaAggPortNumListPortIdExist(p.AggId, pId) {
		p.LaAggPortDisable()
	}
}

func EnableLaAggPort(pId uint16) {
	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) &&
		p.aggSelected == LacpAggUnSelected &&
		LaAggPortNumListPortIdExist(p.AggId, pId) {
		p.LaAggPortEnabled()

		// TODO: NEED METHOD to get link status
		p.LinkOperStatus = true

		if p.LinkOperStatus &&
			p.aggSelected == LacpAggUnSelected {
			p.checkConfigForSelection()
		}
	}
}

// SetLaAggPortLacpMode will set the various
// lacp modes - On, Active, Passive
func SetLaAggPortLacpMode(pId uint16, mode int) {

	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		prevMode := LacpModeGet(p.ActorOper.State, p.lacpEnabled)
		p.LaPortLog(fmt.Sprintf("PrevMode", prevMode, "NewMode", mode))

		// Update the transmission mode
		if mode != prevMode &&
			mode == LacpModeOn {
			p.LaAggPortLacpDisable()

			// Actor/Partner Aggregation == true
			// agg individual
			// partner admin Key, port == actor admin Key, port
			if p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateDetached ||
				p.MuxMachineFsm.Machine.Curr.CurrentState() == LacpMuxmStateCDetached {
				// lets check for selection
				p.checkConfigForSelection()
			}

		} else if mode != prevMode &&
			prevMode == LacpModeOn {
			p.LaAggPortLacpEnabled(mode)
		} else if mode != prevMode {
			if mode == LacpModeActive {
				LacpStateSet(&p.actorAdmin.State, LacpStateActivityBit)
				// must also set the operational State
				LacpStateSet(&p.ActorOper.State, LacpStateActivityBit)
			} else {
				LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
				// must also set the operational State
				LacpStateClear(&p.ActorOper.State, LacpStateActivityBit)
			}
		}
	}
}

// SetLaAggPortLacpPeriod will set the periodic rate at which a packet should
// be transmitted.  What this actually means is at what rate the peer should
// transmit a packet to us.
// FAST and SHORT are the periods, the lacp state timeout is encoded such
// that FAST  is 1 and SHORT is 0
func SetLaAggPortLacpPeriod(pId uint16, period time.Duration) {

	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		rxm := p.RxMachineFsm
		p.LaPortLog(fmt.Sprintf("NewPeriod", period))

		// lets set the period
		if period == LacpFastPeriodicTime {
			LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateClear(&p.ActorOper.State, LacpStateTimeoutBit)
		}
		if timeoutTime, ok := rxm.CurrentWhileTimerValid(); !ok {
			rxm.CurrentWhileTimerTimeoutSet(timeoutTime)
		}
	}
}

// SetLaAggPortLacpTimeout will set the timeout at which a packet should
// declare a link down.
// SHORT and LONG are the periods, the lacp state timeout is encoded such
// that SHORT is 3 * FAST PERIODIC TIME and LONG is 3 * SLOW PERIODIC TIME
func SetLaAggPortLacpTimeout(pId uint16, timeout time.Duration) {

	var p *LaAggPort

	// port exists
	// port is unselected
	// agg exists
	if LaFindPortById(pId, &p) {
		rxm := p.RxMachineFsm
		// lets set the period
		if timeout == LacpShortTimeoutTime {
			LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)
		} else {
			LacpStateClear(&p.actorAdmin.State, LacpStateTimeoutBit)
			// must also set the operational State
			LacpStateClear(&p.ActorOper.State, LacpStateTimeoutBit)
		}
		if timeoutTime, ok := rxm.CurrentWhileTimerValid(); !ok {
			rxm.CurrentWhileTimerTimeoutSet(timeoutTime)
		}
	}
}

func AddLaAggPortToAgg(aggId int, pId uint16) {

	var a *LaAggregator
	var p *LaAggPort

	// both add and port must have existed
	if LaFindAggById(aggId, &a) && LaFindPortById(pId, &p) &&
		p.aggSelected == LacpAggUnSelected &&
		!LaAggPortNumListPortIdExist(aggId, pId) {

		p.LaPortLog(fmt.Sprintf("Adding LaAggPort %d to LaAgg %d", pId, aggId))
		// add port to port number list
		a.PortNumList = append(a.PortNumList, p.PortNum)
		// add reference to aggId
		p.AggId = aggId

		// attach the port to the aggregator
		//LacpStateSet(&p.actorAdmin.State, LacpStateAggregationBit)

		// Port is now aggregatible
		//LacpStateSet(&p.ActorOper.State, LacpStateAggregationBit)

		// well obviously this should pass
		//p.checkConfigForSelection()
	}
}

func DeleteLaAggPortFromAgg(aggId int, pId uint16) {

	var a *LaAggregator
	var p *LaAggPort

	// both add and port must have existed
	if LaFindAggById(aggId, &a) && LaFindPortById(pId, &p) &&
		p.aggSelected == LacpAggSelected &&
		LaAggPortNumListPortIdExist(aggId, pId) {

		// detach the port from the agg port list
		for idx, PortNum := range a.PortNumList {
			if PortNum == pId {
				a.PortNumList = append(a.PortNumList[:idx], a.PortNumList[idx+1:]...)
			}
		}

		LacpStateClear(&p.actorAdmin.State, LacpStateAggregationBit)

		// if port is enabled and lacp is enabled
		p.LaAggPortDisable()

		// Port is now aggregatible
		//LacpStateClear(&p.ActorOper.State, LacpStateAggregationBit)
		// inform mux machine of change of State
		// unnecessary as rx machine should set unselected to mux
		//p.checkConfigForSelection()
	}
}

func GetLaAggPortActorOperState(pId uint16) uint8 {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		return p.ActorOper.State
	}
	return 0
}

func GetLaAggPortPartnerOperState(pId uint16) uint8 {
	var p *LaAggPort
	if LaFindPortById(pId, &p) {
		return p.PartnerOper.State
	}
	return 0
}
