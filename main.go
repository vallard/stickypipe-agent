/* stickypipe agent
Takes environment variables:
	SWITCHES="192.168.30.1,c2960g,nexus5k-top"
	COMMUNITY="public"
	CONSUMERSECRET="Bof7aire/Pangeib8yaxum4Ai"
	CONSUMERID="Ah1IezaiYaf3Tau3Eig4quaijie0Ik7R"

then pipes the output up to sitckypipe in a JSON based string:
{ name: c2960g, interface-id: 1733017, interface-name: GigabitEthernet0/10, interface-in: 3866362551, interface-out: 345343003, timestamp: 1438023632  }


*/
package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/alouca/gosnmp"
)

type ifMessage struct {
	switchName string
	id         string
	name       string
	in         int
	out        int
	ts         string
}

var wg sync.WaitGroup
var mutex sync.Mutex

func handleError(err error) {
	fmt.Println("error in output")
	log.Fatal(err)
}

//func walkValue(s *gosnmp.GoSNMP, oid string, key string, m map[string]map[string]string) {
func walkValue(oid string, key string, m map[string]map[string]string) {
	s, err := gosnmp.NewGoSNMP("10.93.234.5", "public", gosnmp.Version2c, 5)
	if err != nil {
		handleError(err)
		return
	}

	if resp, err := s.Walk(oid); err != nil {
		handleError(err)
	} else {
		for _, pdu := range resp {
			oidbits := strings.Split(pdu.Name, ".")
			ifIndex := oidbits[len(oidbits)-1]
			value := ""
			switch pdu.Type {
			case gosnmp.OctetString:
				value = pdu.Value.(string)
			case gosnmp.Counter32:
				value = fmt.Sprintf("%d", pdu.Value.(int))
			case gosnmp.Counter64:
				value = fmt.Sprintf("%d", pdu.Value.(int64))
			case gosnmp.Gauge32:
				value = fmt.Sprintf("%d", pdu.Value.(int))
			default:
				value = "decode this"
			}
			fmt.Printf("%s / %s\n", key, value)
			mutex.Lock()
			{
				if m[ifIndex] != nil {
					m[ifIndex][key] = value
				} else {
					m[ifIndex] = map[string]string{key: value}
				}
			}
			mutex.Unlock()
		}
	}

}

func getHostname(s *gosnmp.GoSNMP) {
	resp, err := s.Get(".1.3.6.1.2.1.1.5.0")
	if err == nil {
		for _, v := range resp.Variables {
			switch v.Type {
			case gosnmp.OctetString:
				log.Printf("Response: %s : %s : %s \n", v.Name, v.Value.(string), v.Type.String())
			}
		}
	} else {
		log.Fatal(err)
	}

}

func main() {

	// All the OIDs we'll snmp walk through to get
	oidWork := map[string]string{
		".1.3.6.1.2.1.1.5.0":       "sysName",
		".1.3.6.1.2.1.2.2.1.2":     "name",
		".1.3.6.1.2.1.2.2.1.10":    "ifInOctets",
		".1.3.6.1.2.1.2.2.1.16":    "ifOutOctets",
		".1.3.6.1.2.1.31.1.1.1.6":  "ifHCInOctets",
		".1.3.6.1.2.1.31.1.1.1.10": "ifHCOutOctets",
		".1.3.6.1.2.1.31.1.1.1.15": "ifHighSpeed",
	}

	// mapping hash table for interface names.
	m := make(map[string]map[string]string)
	wg.Add(len(oidWork))

	// concurrently execute all of the snmp walks
	for oid, name := range oidWork {
		go func(o string, n string) {
			defer wg.Done()
			walkValue(o, n, m)
		}(oid, name)
	}

	// wait for all the snmpwalks to finish.
	wg.Wait()

	// now we have all the walks, let's push it up.
	for k, v := range m {
		log.Printf("Key: %s / Value: %s", k, v)
	}

}
