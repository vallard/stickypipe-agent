# stickypipe-agent
SNMP agent written in Go to collect information from network switch and send to stickypipe

## Configure SNMP on your switches

### Cisco 2960 
example to configure SNMP v2
```
en
conf t
snmp-server community public ro 
```

## Research

http://tinyurl.com/6zlevq
http://www.cisco.com/c/en/us/support/docs/ip/simple-network-management-protocol-snmp/26007-faq-snmpcounter.html

for ifSpeed if its greater than what's supported than the maximum value of 
4,294,967,295 then ifHighSpeed should be used.  Therefore we are using ifHighSpeed. 
This is 1.3.6.1.2.31.1.1.1.15
http://tools.cisco.com/Support/SNMP/do/BrowseOID.do?local=en&translate=Translate&objectInput=1.3.6.1.2.1.31.1.1.1.15
