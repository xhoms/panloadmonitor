package main

import (
	"encoding/csv"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/xhoms/gopanosapi"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	SHOWDEVICEGROUPS = "<show><devicegroups></devicegroups></show>"
)

func trueIfErr(apiC gopanosapi.ApiConnector, err error) bool {
	if err != nil {
		log.Fatal(err.Error())
		return true
	}
	if apiC.LastStatus == gopanosapi.STATUS_ERROR {
		log.Fatal(apiC.LastResponseMessage)
		return true
	}
	return false
}

type dgs struct {
	DG []struct {
		Devices []struct {
			Serial    string `xml:"serial"`
			Connected string `xml:"connected"`
			SwVersion string `xml:"sw-version"`
		} `xml:"devices>entry"`
	} `xml:"entry"`
}

func (d *dgs) getSerials() map[string]string {
	serials := make(map[string]string)
	for _, dg := range d.DG {
		for _, dev := range dg.Devices {
			if dev.Connected == "no" {
				continue
			}
			if _, ok := serials[dev.Serial]; !ok {
				serials[dev.Serial] = dev.SwVersion
			}
		}
	}
	return serials
}

type worker struct {
	apiconn     gopanosapi.ApiConnector
	outDir      string
	interactive bool
	tick        *time.Ticker
}

func (w *worker) interactivePrint(msg string) {
	if w.interactive {
		fmt.Println(msg)
	}
}

func (w *worker) writeCsv(data [][]string, fileName string) {
	csvFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(0)
	}
	wf := csv.NewWriter(csvFile)
	wf.WriteAll(data)
	csvFile.Close()
}

func (w *worker) do(isPanorama, loop bool) {
	now := time.Now()
	if loop {
		w.tick = time.NewTicker(24 * time.Hour)
	}
	for {
		fPrefix := fmt.Sprintf("%4d%02d%02d", now.Year(), now.Month(), now.Day())
		w.interactivePrint(fmt.Sprintf("Sample prefix will be %v", fPrefix))
		var csvSample [][]string
		switch isPanorama {
		case true:
			// Step 1: Let's get the list of managed devices
			w.interactivePrint("Attempting to get the list of connected devices")
			dgdata, dgerr := w.apiconn.Op(SHOWDEVICEGROUPS)
			if !trueIfErr(w.apiconn, dgerr) {
				deviceGroups := dgs{}
				serialErr := xml.Unmarshal(dgdata, &deviceGroups)

				// Something is wrong parsing the Panorama xml response
				if serialErr != nil {
					fmt.Println(serialErr)
					os.Exit(1)
				}

				// Loop all serials to get performance data

				for serial, swVersion := range deviceGroups.getSerials() {
					w.interactivePrint(fmt.Sprintf("Switching to device serial number %v", serial))
					w.apiconn.SetTarget(serial)
					csvSample = DataProc(w.apiconn, swVersion)
					outFileName := filepath.Join(w.outDir, fPrefix+"_"+serial+".csv")
					w.interactivePrint(fmt.Sprintf("Saving %v", outFileName))
					w.writeCsv(csvSample, outFileName)
				}
				w.apiconn.SetTarget("")
			}
		case false:
			csvSample = DataProc(w.apiconn, w.apiconn.PanosVersion)
			outFileName := filepath.Join(w.outDir, fPrefix+".csv")
			w.interactivePrint(fmt.Sprintf("Saving %v", outFileName))
			w.writeCsv(csvSample, outFileName)
		}
		if loop {
			w.interactivePrint("Going to sleep until next tick ...")
			now = <-w.tick.C
		} else {
			break
		}
	}
}

func main() {
	var debug = flag.Bool("d", false, "Generate debug traces in STDERR")
	var host = flag.String("h", "", "Hostname or IP Address")
	var username = flag.String("u", "", "Username")
	var password = flag.String("p", "", "Password")
	var apikey = flag.String("k", "", "API Key")
	var helpNeeded = flag.Bool("help", false, "Show this help message")
	var interactive = flag.Bool("i", false, "provide interactive (non cron) stepped information")
	var outDir = flag.String("dir", "", "Output directory")
	var isPanorama = flag.Bool("panorama", false, "Flag to loop through all devices connected to Panorama")
	var loop = flag.Bool("loop", false, "Keep panloadmonitor running with a sample each 24h")
	var err error
	flag.Parse()
	var wrkr worker
	wrkr.apiconn.Debug(*debug)
	if *debug {
		log.Println("Starting Program...")
	}
	if *helpNeeded {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *host == "" {
		fmt.Println("Error: Provide -h (hostname) of the PANOS device. Use -help for help")
		os.Exit(0)
	}
	if *apikey == "" && *username == "" {
		fmt.Println("Error: Either -u (user) and -p (password) or -k (api key) must be provided. Use -help for help")
		os.Exit(0)
	} else if *apikey == "" {
		if *password == "" {
			fmt.Println("Error: both -u (user) and -p (password) must be provided")
			os.Exit(0)
		}
		wrkr.apiconn.Init(*host)
		err = wrkr.apiconn.Keygen(*username, *password)
		if err != nil {
			log.Fatal(err.Error())
			os.Exit(0)
		}
	} else {
		wrkr.apiconn.Init(*host)
		err = wrkr.apiconn.SetKey(*apikey)
		if err != nil {
			log.Fatal(err.Error())
			os.Exit(0)
		}
	}

	if *outDir != "" {
		stat, dirErr := os.Stat(*outDir)
		if dirErr == nil {
			if !stat.IsDir() {
				fmt.Printf("Error: %v exists but it is not a directory.\n", *outDir)
				os.Exit(0)
			}
		} else {
			fmt.Printf("Error: generic error opening output directory\n%v\n", dirErr)
			os.Exit(0)
		}
	}

	wrkr.interactive = *interactive
	wrkr.outDir = *outDir
	wrkr.do(*isPanorama, *loop)
}
