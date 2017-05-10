package main

import (
	"encoding/xml"
	"github.com/xhoms/gopanosapi"
	"strconv"
	"strings"
)

type reportStruct struct {
	Entry []struct {
		HourUnparsed string `xml:"hour-of-receive_time"`
		Nbytes       string `xml:"nbytes"`
	} `xml:"entry"`
	data map[int]uint64
}

func (rdata *reportStruct) processData() {
	rdata.data = make(map[int]uint64, 24)
	for _, v := range rdata.Entry {
		frag := strings.Split(v.HourUnparsed, ":")[0]
		hourInt, _ := strconv.Atoi(frag[len(frag)-2:])
		nbytes, _ := strconv.ParseUint(v.Nbytes, 10, 64)
		if v, ok := rdata.data[hourInt]; ok {
			rdata.data[hourInt] = v + nbytes
		} else {
			rdata.data[hourInt] = nbytes
		}
	}
}

type sysInfoStruct struct {
	Model string `xml:"model"`
	Time  string `xml:"time"`
	hour  int
}

func (sinfo *sysInfoStruct) processData() {
	frag := strings.Split(sinfo.Time, ":")[0]
	sinfo.hour, _ = strconv.Atoi(frag[len(frag)-2:])
}

type dataProcsStruct struct {
	DP struct {
		DPN []struct {
			XMLName xml.Name
			Entry   []struct {
				Value string `xml:"value"`
			} `xml:"hour>cpu-load-average>entry"`
		} `xml:",any"`
	} `xml:"data-processors"`
	arrayValues                 [60][20][60]uint8
	dps, cores, samples, takenH int
}

func (dproc *dataProcsStruct) processData(hour int) {
	dproc.takenH = hour
	for k1, v1 := range dproc.DP.DPN {
		dproc.dps = k1 + 1
		for k2, v2 := range v1.Entry {
			dproc.cores = k2 + 1
			for k3, v3 := range strings.Split(v2.Value, ",") {
				dproc.samples = k3 + 1
				v3i, _ := strconv.Atoi(v3)
				dproc.arrayValues[k3][k1][k2] = uint8(v3i)
			}
		}
	}
}

func (dproc *dataProcsStruct) sampleByHour(hour, firstCore int) float64 {
	var idx, loadSum int
	if hour > dproc.takenH {
		idx = 24 + dproc.takenH - hour
	} else {
		idx = dproc.takenH - hour
	}
	for dataPlane := 0; dataPlane < dproc.dps; dataPlane++ {
		for coreNum := firstCore; coreNum < dproc.cores; coreNum++ {
			loadSum += int(dproc.arrayValues[idx][dataPlane][coreNum])
		}
	}
	return float64(loadSum) / float64((dproc.dps * (dproc.cores - firstCore)))
}

const SHOWRUNNINGPROC = "<show><running><resource-monitor><hour></hour></resource-monitor></running></show>"
const SHOWSYSTEMINFO = "<show><system><info></info></system></show>"
const REPORTALL = "<type><appstat><aggregate-by><member>hour-of-receive_time</member></aggregate-by><values><member>nbytes</member></values></appstat></type><period>last-24-hrs</period><topn>25</topn><topm>10</topm>"

func DataProc(apiC gopanosapi.ApiConnector) (csvData [][]string) {
	csvData = make([][]string, 25)
	csvData[0] = []string{"hour", "dpload", "mbps"}
	// let's get the show system info data
	var sinfo sysInfoStruct
	response, err := apiC.Op(SHOWSYSTEMINFO)
	trueIfErr(apiC, err)
	xml.Unmarshal(response, &sinfo)
	sinfo.processData()

	// let's get the show running process data
	response, err = apiC.Op(SHOWRUNNINGPROC)
	if trueIfErr(apiC, err) {
		return
	}
	var values dataProcsStruct
	xml.Unmarshal(response, &values)
	values.processData(sinfo.hour)

	var rdataAll reportStruct
	// let's get the allApps data
	response, err = apiC.Report(gopanosapi.REPORT_DYNAMIC, "", REPORTALL)
	if trueIfErr(apiC, err) {
		return
	}
	xml.Unmarshal(response, &rdataAll)
	rdataAll.processData()

	paModel := sinfo.Model
	firstCore := 0
	if paModel == "PA-200" || paModel == "PA-VM" {
		firstCore = 1
	}
	for h := 0; h < 24; h++ {
		csvData[h+1] = make([]string, 3)
		allData, hasAllData := rdataAll.data[h]
		fallData := float64(allData)
		if values.sampleByHour(h, firstCore) != 0 && hasAllData {
			mbps := fallData / 360000000.0
			dpl := values.sampleByHour(h, firstCore)
			csvData[h+1][0] = strconv.Itoa(h)
			csvData[h+1][1] = strconv.FormatFloat(dpl, 'f', 2, 64)
			csvData[h+1][2] = strconv.FormatFloat(mbps, 'f', 2, 64)
		} else {
			csvData[h+1] = []string{strconv.Itoa(h), "0.00", "0.00"}
		}
	}
	return
}
