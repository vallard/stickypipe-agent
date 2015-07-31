// Simple tool to get infromation from Nexus 9k
// I used this to test before merging it into the other program.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/joeshaw/envdecode"
)

type ins_api struct {
	version string
	sid     string
	outputs map[string]interface{}
}

func main() {
	var params struct {
		Endpoints   string `env:"SP_ENDPOINTS,required"`
		Credentials string `env:"SP_ENDPOINT_CREDENTIALS,required"`
	}

	if err := envdecode.Decode(&params); err != nil {
		log.Fatalln(err)
	}

	endpoints := strings.Split(params.Endpoints, ",")
	// creds should be of the form: user:password,user:password
	cred := strings.Split(params.Credentials, ",")
	if len(endpoints) != len(cred) {
		log.Fatal("Each endpoint should have a corresponding credential")
	}

	for i, ep := range endpoints {
		up := strings.Split(cred[i], ":")
		if len(up) < 2 {
			log.Fatal("Credentials must be of form user:password")
			break
		}
		var jsonStr = []byte(`{
			"ins_api": {
					"version":       "1.0",
					"type":          "cli_show",
					"chunk":         "0",
					"sid":           "1",
					"input":         "sh interface counters",
					"output_format": "json",
				}
			}
				`)
		req, err := http.NewRequest("POST", "http://"+ep+"/ins", bytes.NewBuffer(jsonStr))
		if err != nil {
			log.Println("HTTP Post: ", err)
		}

		req.Header.Set("content-type", "application/json")
		req.SetBasicAuth(up[0], up[1])

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("response error: ", err)
		}
		defer resp.Body.Close()

		fmt.Println("response Status: ", resp.Status)
		fmt.Println("response Headers: ", resp.Header)
		body, err := ioutil.ReadAll(resp.Body)
		//rr := make(map[string]interface{})
		rr := ins_api{}
		err = json.Unmarshal(body, &rr)
		if err != nil {
			log.Fatal("Error unmarshalling: ", err)
		}
		//fmt.Println("responseBody:", string(body))

		fmt.Println(rr)
	}

	log.Println("End of program")
}
