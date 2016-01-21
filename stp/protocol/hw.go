// hw.go
package lacp

import (
	hwconst "asicd/asicdConstDefs"
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"strconv"
	"strings"
	"utils/ipcutils"
)

type LACPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	LACPClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var asicdclnt AsicdClient

// look up the various other daemons based on c string
func GetClientPort(paramsFile string, c string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		fmt.Println("Error in reading configuration file:%s err:%s", paramsFile, err)
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		fmt.Println("Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == c {
			return client.Port
		}
	}
	return 0
}

// connect the the asic d
func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {
		fmt.Printf("found asicd at port %d\n", port)
		asicdclnt.Address = "localhost:" + strconv.Itoa(port)
		asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
		if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
			fmt.Println("connecting to asicd\n")
			asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
			asicdclnt.IsConnected = true
		}
	}
}

// convert the lacp port names name to asic format string list
func asicDPortBmpFormatGet(distPortList []string) string {
	s := ""
	dLength := len(distPortList)

	for i := 0; i < dLength; i++ {
		num := strings.Split(distPortList[i], "-")[1]
		if i == dLength-1 {
			s += num
		} else {
			s += num + ","
		}
	}
	return s

}

func asicdGetPortLinkStatus(intfNum string) bool {

	if asicdclnt.ClientHdl != nil {
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortConfig(hwconst.MIN_SYS_PORTS, hwconst.MIN_SYS_PORTS)
		if err == nil && bulkInfo.ObjCount != 0 {
			objCount := int64(bulkInfo.ObjCount)
			for i := int64(0); i < objCount; i++ {
				if bulkInfo.PortConfigList[i].Name == intfNum {
					return bulkInfo.PortConfigList[i].OperState == pluginCommon.UpDownState[1]
				}
			}
		}
		fmt.Printf("asicDGetPortLinkSatus: could not get status for port %s, failure in get method\n", intfNum)
	}
	return true

}