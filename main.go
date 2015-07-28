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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
			//fmt.Printf("%s / %s\n", key, value)
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
		".1.3.6.1.2.1.1.5":         "sysName",
		".1.3.6.1.2.1.2.2.1.2":     "name",
		".1.3.6.1.2.1.2.2.1.10":    "ifInOctets",
		".1.3.6.1.2.1.2.2.1.16":    "ifOutOctets",
		".1.3.6.1.2.1.31.1.1.1.6":  "ifHCInOctets",
		".1.3.6.1.2.1.31.1.1.1.10": "ifHCOutOctets",
		".1.3.6.1.2.1.31.1.1.1.15": "ifHighSpeed",
	}

	// The main waitgroup for each switch waits for
	// each switch to finish its job.
	var mainWg sync.WaitGroup
	// The array of waitgroups are the jobs that each
	// switch must proccess.
	wg := make([]sync.WaitGroup, len(endpoints))

	// we will run in a continuous loop forever!
	// or at least until the user hits ctrl-c or we get a signal interrupt.
	stop := false
	signalChan := make(chan os.Signal, 1)

	go func() {
		<-signalChan
		stop = true
		// TODO: This delay really sucks.  Fix this using a channel.
		log.Println("Cleaning up... this could take 60 seconds.  Sorry.  Please be patient")
		// send stuff on the channel until it closes.

	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {

		if stop {
			break
		}
		// make sure we wait for each of the switches
		mainWg.Add(len(endpoints))

		// go through each device and grab the counters.
		for i, endpoint := range endpoints {
			m := make(map[string]map[string]string)
			go func(e string, c string, w sync.WaitGroup) {
				defer mainWg.Done()
				// mapping hash table for interface names.
				w.Add(len(oidWork))
				// concurrently execute all of the snmp walks
				for oid, name := range oidWork {
					go func(o string, n string) {
						defer w.Done()
						walkValue(e, c, o, n, m)
					}(oid, name)
				}
				w.Wait()
				processCollectedData(m)
			}(endpoint, creds[i], wg[i])
		}
		// wait for all the snmpwalks to finish.
		mainWg.Wait()

		// now sleep for a while and then run again.
		timeoutchan := make(chan bool)
		go func() {
			fmt.Println("Sleeping for 60 seconds...")
			<-time.After(60 * time.Second)
			timeoutchan <- true
		}()
		select {
		case <-timeoutchan:
			break
		}
	}
}

// take all the data we were given and format it to JSON to send up
// to the server.
func processCollectedData(m map[string]map[string]string) {
	// get the name of the switch:
	sw := m["0"]["sysName"]
	// get the timestamp
	now := time.Now()

	sendMap := make(map[string][]string)
	var sendStrings []string
	//go through each switch for k, v := range m {
	for k, v := range m {
		sendMe := fmt.Sprintf("'switch':'%s', 'ifName':'%s','timeStamp':'%d','ifInOctets':'%s', 'ifHCInOctets': '%s', 'ifOutOctets': '%s', 'ifHCOutOctets': '%s', 'ifHighSpeed': '%s', 'ifId': '%s'\n",
			sw,
			v["name"],
			now.Unix(),
			v["ifInOctets"],
			v["ifHCInOctets"],
			v["ifOutOctets"],
			v["ifHCOutOctets"],
			v["ifHighSpeed"],
			k)

		sendStrings = append(sendStrings, sendMe)
	}
	sendMap[sw] = sendStrings
	fmt.Println(sendMap)
}
