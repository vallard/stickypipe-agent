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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joeshaw/envdecode"
	"github.com/vallard/gosnmp"
	"github.com/vallard/stickypipe-agent/nxapi"
)

var mutex sync.Mutex

func handleError(err error) {
	fmt.Println("error:", err)
}

/* Get NXAPI information
Arguments:
 server - Nexus Switch (10.93.234.2, sw001, or something reachable)
 creds - user/password pair that looks like admin:cisco (notice no :'s are supported in the password.)
 map - map we want to store this stuff.
*/

func getNXAPIData(server string, command string, outputName string, creds string, m map[string]map[string]map[string]string) {
	// argument for string looks like: admin:cisco where admin is the user and cisco is the password.
	up := strings.Split(creds, ":")
	// make sure that we parsed the username and password.
	if len(up) < 2 {
		log.Fatal("Credentials must be of form user:password")
		return
	}
	// The command we run to get the port interface statistics.
	/*
		nxcmd := NewNXAPIPost(command)
		fmt.Println(nxcmd)
		jsonStr, err := json.Marshal(nxcmd)
	*/
	var jsonStr = []byte(`{
					"ins_api": {
							"version":       "1.0",
							"type":          "cli_show",
							"chunk":         "0",
							"sid":           "1",
							"input":         "` + command + `",
							"output_format": "json",
						}
					}
						`)
	// Start formatting our HTTP POST request.
	req, err := http.NewRequest("POST", "http://"+server+"/ins", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("HTTP Post: ", err)
	}
	// The header has to be set to application/json
	req.Header.Set("content-type", "application/json")
	// add the username and password to the header.
	req.SetBasicAuth(up[0], up[1])

	// create a new http client to execute the request.
	client := &http.Client{}
	// execute the request.
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("response error: ", err)
	}
	defer resp.Body.Close()

	//fmt.Println("response Status: ", resp.Status)
	//fmt.Println("response Headers: ", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	//fmt.Println("responseBody:", string(body))
	var rr nxapi.NXAPI_Response
	//var rr interface{}
	err = json.Unmarshal(body, &rr)
	if err != nil {
		log.Fatal("Error unmarshalling: ", err)
	}
	// Print out the raw string to debug.
	for _, b := range rr.Ins_api.Outputs {
		if b.Input == "show version" {
			h, ok := b.Body["host_name"].(string)
			if !ok {
				continue
			}
			mutex.Lock()
			{
				if m[server] != nil {
					m[server]["hostname"] = map[string]string{"host_name": "bar"}
				} else {
					m[server] = map[string]map[string]string{"hostname": map[string]string{"host_name": h}}
				}
			}
			mutex.Unlock()
		} else if b.Input == "show interface counters" {
			interfaceCounters := nxapi.NewInterfaceCounters(b.Body)
			fmt.Println(interfaceCounters)
			//m[server]
		}
	}
	//fmt.Println(m[server])
}

/* walkvalues:
 Arguments:
	server - like a switch (10.93.234.2, or c2960-001, or something like that. )
  creds - this is the SNMP community string (public, 234jfj23vjA233kj3, etc)
	oid - the OID we're going to walk through
	key - the key we want to store the individual values
	map - the map that we want to store this in.
*/
func walkValue(server string, creds string, oid string, key string, m map[string]map[string]map[string]string) {
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
				if m[server][ifIndex] != nil {
					m[server][ifIndex][key] = value
				} else {
					m[server][ifIndex] = map[string]string{key: value}
				}
			}
			mutex.Unlock()
		}
	}
	s.Close()

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

	// all the commands we walk through NXAPI to get.
	nxapiWork := map[string]string{
		"show version":            "version",
		"show interface counters": "counters",
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
	interruptChan := make(chan struct{}, 1)
	go func() {
		<-signalChan
		stop = true
		log.Println("Cleaning up...")
		// send stuff to the channel so that it closes.  That way we don't have to wait so long.
		interruptChan <- struct{}{}
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
			// figure out which method to run:
			em := strings.Split(endpoint, ":")
			if len(em) < 2 {
				fmt.Println("Invalid input: ", endpoint)
				fmt.Println("please export SP_ENDPOINTS=<name>:<method> where method is SNMP or NXAPI")
				// don't wait for me any more.
				mainWg.Add(-1)
				// go to the next switch
				continue
			}
			/* big map:
			server {
				interface {
					key : value
				}
			}
			*/
			m := make(map[string]map[string]map[string]string)
			if em[1] == "SNMP" {
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
					processCollectedSNMPData(m[e])
				}(em[0], creds[i], wg[i])
			} else if em[1] == "NXAPI" {
				// Yes.. this is hard to process, so let's walk through this.
				// If this switch is an NXAPI endpoint, we are going to kick off a goroutine
				// This go routine is going to call several commands against the switch.
				go func(e string, cre string, w sync.WaitGroup) {
					// When we finish processing all the commands against this switch, tell the main workGroup we are done.
					defer mainWg.Done()

					// we need to add a waitgroup for this task.
					// This waitgroup is specific for this switch.
					// We dont' want to wait for all the other switches to finish processing before
					// sending the data to the cloud
					w.Add(len(nxapiWork))
					// Go through each command that we want to process.
					for cmd, name := range nxapiWork {
						// kick off a go routine for each of the commands we want to get
						go func(c string, outputName string) {
							// make sure we decrement the switch waitgroup.
							defer w.Done()
							// get the data.  This is where the work takes place.
							getNXAPIData(e, c, outputName, cre, m)
						}(cmd, name)
					}
					// wait for all the switch waitgroups to finish.
					w.Wait()
					// now we have all the data for this switch, let's process it.
					processCollectedNXAPIData(m[e])
				}(em[0], creds[i], wg[i])
			}
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
		case <-interruptChan:
			// close the channel so other threads will stop blocking and finish.
			close(interruptChan)
			break
		}
	}
}

// process NXAPI data
func processCollectedNXAPIData(m map[string]map[string]string) {
}

// take all the data we were given and format it to JSON to send up
// to the server.
func processCollectedSNMPData(m map[string]map[string]string) {
	// get the name of the switch:
	sw := m["0"]["sysName"]
	// get the timestamp
	now := time.Now()

	sendMap := make(map[string][]string)
	var sendStrings []string
	//go through each switch for k, v := range m {
	for k, v := range m {
		// don't send if there is no data to send.
		if emptyValues(v) {
			continue
		}
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
	// Todo: Send this up to the main server.
	fmt.Println(sendMap)
}

// see if any of the values are empty.
func emptyValues(v map[string]string) bool {
	if v["name"] == "" {
		return true
	}
	if v["ifInOctets"] == "" {
		return true
	}
	if v["ifHCInOctets"] == "" {
		return true
	}
	if v["ifOutOctets"] == "" {
		return true
	}
	if v["ifHCOutOctets"] == "" {
		return true
	}
	if v["ifHighSpeed"] == "" {
		return true
	}
	return false
}
