// debugEventLog this code is meant to serialize the logging States
package lacp

import (
	//"fmt"
	"strings"
	//"time"
	"log/syslog"
)

type LacpDebug struct {
	LacpLogChan chan string
	logger      *syslog.Writer
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpDebug() *LacpDebug {
	logger, _ := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "LACP")
	lacpdebug := &LacpDebug{
		LacpLogChan: make(chan string, 100),
		logger:      logger,
	}

	return lacpdebug
}

func (l *LacpDebug) Stop() {
	close(l.LacpLogChan)
}

func (p *LaAggPort) LacpDebugEventLogMain() {

	p.LacpDebug = NewLacpDebug()

	go func(port *LaAggPort) {

		for {
			select {

			case msg, logEvent := <-port.LacpDebug.LacpLogChan:
				if logEvent {
					port.LacpDebug.logger.Info(strings.Join([]string{p.IntfNum, msg}, "-"))
				} else {
					return
				}
			}
		}
	}(p)
}

func (txm *LacpTxMachine) LacpTxmLog(msg string) {
	if txm.Machine.Curr.IsLoggerEna() {
		txm.log <- strings.Join([]string{"TXM", msg}, ":")
	}
}

func (cdm *LacpCdMachine) LacpCdmLog(msg string) {
	if cdm.Machine.Curr.IsLoggerEna() {
		cdm.log <- strings.Join([]string{"CDM", msg}, ":")
	}
}

func (ptxm *LacpPtxMachine) LacpPtxmLog(msg string) {
	if ptxm.Machine.Curr.IsLoggerEna() {
		ptxm.log <- strings.Join([]string{"PTXM", msg}, ":")
	}
}

func (rxm *LacpRxMachine) LacpRxmLog(msg string) {
	if rxm.Machine.Curr.IsLoggerEna() {
		rxm.log <- strings.Join([]string{"RXM", msg}, ":")
	}
}

func (muxm *LacpMuxMachine) LacpMuxmLog(msg string) {
	if muxm.Machine.Curr.IsLoggerEna() {
		muxm.log <- strings.Join([]string{"MUXM", msg}, ":")
	}
}
