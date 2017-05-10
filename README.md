# panloadmonitor
Small tool to get last 24hour performance metrics of panos devices
## installation
Use either the binaries in https://github.com/xhoms/panloadmonitor/tree/master/bin or get the source into your GO workspace
```
go get github.com/xhoms/panloadmonitor
```
## usage
Basic usage requires just the following command line switches
* -h (host) : either in FQDN or IP address format
* -k (key) : API key to be used.
* -u (user) : you can provide user and password instead of API key
* -p (password)
* -i (interactive) : provide information about the progress in stdout

Example of basic usage
```
$ ./panloadmonitor -h 172.16.214.199 -u admin -p admin -i
Sample prefix will be 20170510
Saving 20170510.csv
```
Other command line switches
* -loop : once a run is completed keep waiting for another run after 24 hours
* -panorama : loop through all devices connected to a Panorama and generate a csv file for each one
* -d (debug) : provide API interaction verbosity

Example of loop panorama usage
```
$ ./panloadmonitor -h 172.16.214.203 -u admin -p admin -i -panorama -loop
Sample prefix will be 20170510
Attempting to get the list of connected devices
Switching to device serial number 007000000021636
Saving 20170510_007000000021636.csv
Switching to device serial number 015300000425
Saving 20170510_015300000425.csv
Going to sleep until next tick ...
```
