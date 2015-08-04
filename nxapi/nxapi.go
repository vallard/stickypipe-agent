package nxapi

/* These structures show the type of response we get back from
The NXAPI.  This could be quite big.  The Output is the same up until the
body.  That is where the outputs differ depending on which command is given.
*/
type NXAPI_Response struct {
	Ins_api Ins_API
}

type Ins_API struct {
	Type    string
	Version string
	Sid     string
	Outputs map[string]Output
}

type Output struct {
	Input string
	Msg   string
	Code  string
	Body  map[string]interface{}
}

type Version struct {
	Header_Str        string
	Bios_Ver_Str      string
	Kickstart_Ver_Str string
	Bios_Cmpl_Time    string
	Kick_File_Name    string
	Kick_Cmpl_Time    string
	Kick_Tmstmp       string
	Chassis_Id        string
	Cpu_Name          string
	Memory            int
	Mem_Type          string
	Proc_Board_Id     string
	Host_Name         string
	Bootflash_Size    int
	Kern_Uptm_Days    int
	Kern_Uptm_Hrs     int
	Kern_Uptm_Min     int
	Kern_Uptm_Secs    int
	Rr_Reason         string
	Rr_Sys_Ver        string
	Rr_Service        string
	Manufacturer      string
}

func NewVersion(m map[string]string) Version {

	v := Version{
		Host_Name: m["host_name"],
	}
	return v
}

// Can this get any uglier?  Why is there no better way to map
// an interface to a complex structure?  Is this because my structure
// is too complicated?  Maybe.

type InterfaceCounters struct {
	RX_Table TABLE_rx_counters
	TX_Table TABLE_tx_counters
}

func NewInterfaceCounters(m map[string]interface{}) InterfaceCounters {
	//fmt.Println("RX TABLE: ", m["TABLE_rx_counters"], "\n\n")
	//fmt.Println("TX Counters: ", m["TABLE_tx_counters"])
	i := InterfaceCounters{
		RX_Table: NewTableRXCounters(m["TABLE_rx_counters"]),
		TX_Table: NewTableTXCounters(m["TABLE_tx_counters"]),
	}
	return i
}

type TABLE_rx_counters struct {
	Row map[string]ROW_rx_counters
}

type TABLE_tx_counters struct {
	Row map[string]ROW_tx_counters
}

func NewTableRXCounters(i interface{}) TABLE_rx_counters {
	r := map[string]ROW_rx_counters{}
	tempHash := map[string]map[string]interface{}{}
	ifaceMap, ok := i.(map[string]interface{})
	if !ok {
		return TABLE_rx_counters{}
	}
	ifaceArray := ifaceMap["ROW_rx_counters"].([]interface{})
	//fmt.Println(ifaceArray)
	for _, arrValue := range ifaceArray {
		a := arrValue.(map[string]interface{})
		//fmt.Println(a)
		iface := a["interface_rx"].(string)
		// Every value has the interface but for whatever reason the
		// n9 returns two hashes of the same interface.
		// therefore we have to go through the whole thing and put it
		// in a hash and then once its all filled out, push it to a structure.
		// Why?  cause Go doesn't let us modify structures unless we right
		// a bunch of Set routines...
		if tempHash[iface] != nil {
			tempHash[iface]["Interface_rx"] = iface
		} else {
			tempHash[iface] = map[string]interface{}{}
			tempHash[iface]["Interface_rx"] = iface
		}

		metrics := map[string]string{
			"eth_inbytes": "Eth_inbytes",
			"eth_inucast": "Eth_inucast",
			"eth_inmcast": "Eth_inmcast",
			"eth_inbcast": "Eth_inbcast",
			"eth_inpkts":  "Eth_inpkts",
		}
		for metric, hashKey := range metrics {
			if val, ok := a[metric]; ok {
				tempHash[iface][hashKey] = val.(float64)
			} else {
				if tempHash[iface][hashKey] == nil {
					tempHash[iface][hashKey] = float64(0)
				}
			}
		}
	}

	// once we're done with all the interfaces:
	for k, v := range tempHash {

		r[k] = ROW_rx_counters{
			Interface_rx: v["Interface_rx"].(string),
			Eth_inbytes:  v["Eth_inbytes"].(float64),
			Eth_inpkts:   v["Eth_inpkts"].(float64),
			Eth_inucast:  v["Eth_inucast"].(float64),
			Eth_inmcast:  v["Eth_inmcast"].(float64),
			Eth_inbcast:  v["Eth_inbcast"].(float64),
		}
	}
	//fmt.Println(ifaceArray)
	return TABLE_rx_counters{
		Row: r,
	}
}

type ROW_rx_counters struct {
	Interface_rx string
	Eth_inbytes  float64
	Eth_inpkts   float64
	Eth_inucast  float64
	Eth_inmcast  float64
	Eth_inbcast  float64
}

// This is lame because this entire function is a copy of the
// other NewTableRXCounters.  This is definitely not DRY.
// I hang my head in shame.
func NewTableTXCounters(i interface{}) TABLE_tx_counters {
	r := map[string]ROW_tx_counters{}
	tempHash := map[string]map[string]interface{}{}
	ifaceMap, ok := i.(map[string]interface{})
	if !ok {
		return TABLE_tx_counters{}
	}
	ifaceArray := ifaceMap["ROW_tx_counters"].([]interface{})
	//fmt.Println(ifaceArray)
	for _, arrValue := range ifaceArray {
		a := arrValue.(map[string]interface{})
		//fmt.Println(a)
		iface := a["interface_tx"].(string)
		// Every value has the interface but for whatever reason the
		// n9 returns two hashes of the same interface.
		// therefore we have to go through the whole thing and put it
		// in a hash and then once its all filled out, push it to a structure.
		// Why?  cause Go doesn't let us modify structures unless we right
		// a bunch of Set routines...
		if tempHash[iface] != nil {
			tempHash[iface]["Interface_tx"] = iface
		} else {
			tempHash[iface] = map[string]interface{}{}
			tempHash[iface]["Interface_tx"] = iface
		}

		metrics := map[string]string{
			"eth_outbytes": "Eth_outbytes",
			"eth_outpkts":  "Eth_outpkts",
			"eth_outucast": "Eth_outucast",
			"eth_outmcast": "Eth_outmcast",
			"eth_outbcast": "Eth_outbcast",
		}
		for metric, hashKey := range metrics {
			if val, ok := a[metric]; ok {
				tempHash[iface][hashKey] = val.(float64)
			} else {
				if tempHash[iface][hashKey] == nil {
					tempHash[iface][hashKey] = float64(0)
				}
			}
		}
	}

	// once we're done with all the interfaces:
	for k, v := range tempHash {

		r[k] = ROW_tx_counters{
			Interface_tx: v["Interface_tx"].(string),
			Eth_outbytes: v["Eth_outbytes"].(float64),
			Eth_outucast: v["Eth_outucast"].(float64),
			Eth_outmcast: v["Eth_outmcast"].(float64),
			Eth_outbcast: v["Eth_outbcast"].(float64),
		}
	}
	//fmt.Println(ifaceArray)
	return TABLE_tx_counters{
		Row: r,
	}
}

type ROW_tx_counters struct {
	Interface_tx string
	Eth_outpkts  float64
	Eth_outbytes float64
	Eth_outucast float64
	Eth_outmcast float64
	Eth_outbcast float64
}
