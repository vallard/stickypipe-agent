# stickypipe-agent
SNMP agent written in Go to collect information from network switch and send to stickypipe
The agent runs every minute to collect stats.  This is probably easiest to run in a Docker
container.  

## Running the Docker Container

Several environment variables are necessary for you to run:

#### SP_ENDPOINTS
Comma seperated list of switches that we want to collect stats from.  
Each switch should be followed with a :<METHOD> tag to tell the agent how to 
connect to it: 

```
SP_ENDPOINTS="10.93.234.2:SNMP,10.93.234.5:SNMP,10.94.238.122:NXAPI"
```

Current Methods:
* SNMP (this is only v2)
* NXAPI


#### SP_ENDPOINT_CREDENTIALS
Comma seperated list of passwords/community strings.  Even if they are all the same
they should match the number SP_ENDPOINTS.  
```
export SP_ENDPOINT_CREDENTIALS="public,public,admin:cisco"
```
In the above example we have two public community strings for SNMP and a user/password for NXAPI

To run the container: 
```
docker run -d -e SP_ENDPOINTS="10.93.234.2:SNMP,10.93.234.5:SNMP" \
       -e SP_ENDPOINT_CREDENTIALS="public,public" \
       --name stickypipe
       vallard/stickypipe-agent
```
To run to test to see the output: 
```
docker run --rm -it -e SP_ENDPOINTS="10.93.234.2:SNMP,10.93.234.5:SNMP,10.93.238.211:NXAPI" \
       -e SP_ENDPOINT_CREDENTIALS="public,public,admin:password" \
       --name stickypipe
       vallard/stickypipe-agent
```
To connect to it while its running
```
docker exec -it stickypipe /bin/bash
```


### Building the Container
Pretty simple... 
```
docker build -t vallard/stickypipe-agent .
```

## Configure SNMP on your switches

The switches will require that you enable SNMP on them so the agent can collect information. 
Currently SNMPv2 is supported.  Other versions haven't been tested but could probably be added
pretty easily in the future. 

### Cisco 2960 
example to configure SNMP v2.  We create Read Only
```
en
conf t
snmp-server community public ro 
```

### Cisco Nexus 5000
example to configure SNMP v2.  We create Read Only
```
en 
conf
snmp-server community public ro
```



## Research

http://tinyurl.com/6zlevq
http://www.cisco.com/c/en/us/support/docs/ip/simple-network-management-protocol-snmp/26007-faq-snmpcounter.html

for ifSpeed if its greater than what's supported than the maximum value of 
4,294,967,295 then ifHighSpeed should be used.  Therefore we are using ifHighSpeed. 
This is 1.3.6.1.2.31.1.1.1.15
http://tools.cisco.com/Support/SNMP/do/BrowseOID.do?local=en&translate=Translate&objectInput=1.3.6.1.2.1.31.1.1.1.15

# change me
