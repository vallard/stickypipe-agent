/* stickypipe agent
Takes environment variables:
  // End points are our devices such as a network switch.
	SP_ENDPOINTS="192.168.30.1,c2960g,nexus5k-top"
	// Credentials are our logins to the endpoints.
	SP_ENDPOINT_CREDENTIALS="public"

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
	"github.com/joeshaw/envdecode"
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
	fmt.Println("error:", err)
}

/* walkvalues:
 Arguments:
	server - like a switch (10.93.234.2, or c2960-001, or something like that. )
  creds - this is the SNMP community string (public, 234jfj23vjA233kj3, etc)
	oid - the OID we're going to walk through
	key - the key we want to store the individual values
	map - the map that we want to store this in.
*/
func walkValue(server string, creds string, oid string, key string, m map[string]map[string]string) {
	s, err := gosnmp.NewGoSNMP(server, creds, gosnmp.Version2c, 5)
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
	// We require each program to have endpoints defined.
	var params struct {
		Endpoints   string `env:"SP_ENDPOINTS,required"`
		Credentials string `env:"SP_ENDPOINT_CREDENTIALS,required"`
	}

	if err := envdecode.Decode(&params); err != nil {
		log.Fatalln(err)
	}

	endpoints := strings.Split(params.Endpoints, ",")
	creds := strings.Split(params.Credentials, ",")
	if len(endpoints) != len(creds) {
		log.Fatal("Each endpoint should have a corresponding credential")
	}

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
	m := make(map[string]map[string]string)

	// go through each device and grab the counters.
	wg.Add(len(endpoints))
	for i, endpoint := range endpoints {
		go func(e string, c string) {
			defer wg.Done()
			// mapping hash table for interface names.
			wg.Add(len(oidWork))
			// concurrently execute all of the snmp walks
			for oid, name := range oidWork {
				go func(o string, n string) {
					defer wg.Done()
					walkValue(e, c, o, n, m)
				}(oid, name)
			}

		}(endpoint, creds[i])
	}
	// wait for all the snmpwalks to finish.
	wg.Wait()
	// now we have all the walks, let's push it up.
	for k, v := range m {
		log.Printf("Key: %s / Value: %s", k, v)
	}
}
