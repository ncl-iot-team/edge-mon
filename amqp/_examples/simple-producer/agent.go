package main

import (
	"github.com/streadway/systemstat"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"runtime"
	"time"
)

var coresToPegPtr *int64

type stats struct {
	startTime time.Time

	// stats this process
	ProcUptime        float64 //seconds
	ProcMemUsedPct    float64
	ProcCPUAvg        systemstat.ProcCPUAverage
	LastProcCPUSample systemstat.ProcCPUSample `json:"-"`
	CurProcCPUSample  systemstat.ProcCPUSample `json:"-"`

	// stats for whole system
	LastCPUSample systemstat.CPUSample `json:"-"`
	CurCPUSample  systemstat.CPUSample `json:"-"`
	SysCPUAvg     systemstat.CPUAverage
	SysMemK       systemstat.MemSample
	LoadAverage   systemstat.LoadAvgSample
	SysUptime     systemstat.UptimeSample

	// bookkeeping
	procCPUSampled bool
	sysCPUSampled  bool
}

type Data struct{
	IdleCPU float32
	IdleMen float32
}

var statis Data

func NewStats() *stats {
	s := stats{}
	s.startTime = time.Now()
	return &s
}

func (s *stats) PrintStats() {
	up, err := time.ParseDuration(fmt.Sprintf("%fs", s.SysUptime.Uptime))
	upstring := "SysUptime Error"
	if err == nil {
		updays := up.Hours() / 24
		switch {
		case updays >= 365:
			upstring = fmt.Sprintf("%.0f years", updays/365)
		case updays >= 1:
			upstring = fmt.Sprintf("%.0f days", updays)
		default: // less than a day
			upstring = up.String()
		}
	}

	fmt.Println("*********************************************************")
	fmt.Printf("go-top - %s  up %s,\t\tload average: %.2f, %.2f, %.2f\n",
		s.LoadAverage.Time.Format("15:04:05"), upstring, s.LoadAverage.One, s.LoadAverage.Five, s.LoadAverage.Fifteen)

	fmt.Printf("Cpu(s): %.1f%%us, %.1f%%sy, %.1f%%ni, %.1f%%id, %.1f%%wa, %.1f%%hi, %.1f%%si, %.1f%%st %.1f%%gu\n",
		s.SysCPUAvg.UserPct, s.SysCPUAvg.SystemPct, s.SysCPUAvg.NicePct, s.SysCPUAvg.IdlePct,
		s.SysCPUAvg.IowaitPct, s.SysCPUAvg.IrqPct, s.SysCPUAvg.SoftIrqPct, s.SysCPUAvg.StealPct,
		s.SysCPUAvg.GuestPct)

	fmt.Printf("Mem:  %9dk total, %9dk used, %9dk free, %9dk buffers\n", s.SysMemK.MemTotal,
		s.SysMemK.MemUsed, s.SysMemK.MemFree, s.SysMemK.Buffers)
	fmt.Printf("Swap: %9dk total, %9dk used, %9dk free, %9dk cached\n", s.SysMemK.SwapTotal,
		s.SysMemK.SwapUsed, s.SysMemK.SwapFree, s.SysMemK.Cached)

	fmt.Println("************************************************************")
	if s.ProcCPUAvg.PossiblePct > 0 {
		cpuHelpText := "[see -help flag to change %cpu]"
		if *coresToPegPtr > 0 {
			cpuHelpText = ""
		}
		fmt.Printf("ProcessName\tRES(k)\t%%CPU\t%%CCPU\t%%MEM\n")
		fmt.Printf("this-process\t%d\t%3.1f\t%2.1f\t%3.1f\t%s\n",
			s.CurProcCPUSample.ProcMemUsedK,
			s.ProcCPUAvg.TotalPct,
			100*s.CurProcCPUSample.Total/s.ProcUptime/float64(1),
			100*float64(s.CurProcCPUSample.ProcMemUsedK)/float64(s.SysMemK.MemTotal),
			cpuHelpText)
		fmt.Println("%CCPU is cumulative CPU usage over this process' life.")
		fmt.Printf("Max this-process CPU possible: %3.f%%\n", s.ProcCPUAvg.PossiblePct)
	}
}

func (s *stats) setStats(){
	fmt.Printf("Cpu(s): %.1f%%id\n", s.SysCPUAvg.IdlePct)
	fmt.Printf("Mem: %9dk free\n", s.SysMemK.MemFree)
	statis = Data{float32(s.SysCPUAvg.IdlePct), float32(s.SysMemK.MemFree)}
}

func getStats() Data{
		return statis
}

func (s *stats) GatherStats(percent bool) {
	s.SysUptime = systemstat.GetUptime()
	s.ProcUptime = time.Since(s.startTime).Seconds()

	s.SysMemK = systemstat.GetMemSample()
	s.LoadAverage = systemstat.GetLoadAvgSample()

	s.LastCPUSample = s.CurCPUSample
	s.CurCPUSample = systemstat.GetCPUSample()

	if s.sysCPUSampled { // we need 2 samples to get an average
		s.SysCPUAvg = systemstat.GetCPUAverage(s.LastCPUSample, s.CurCPUSample)
	}
	// we have at least one sample, subsequent rounds will give us an average
	s.sysCPUSampled = true

	s.ProcMemUsedPct = 100 * float64(s.CurProcCPUSample.ProcMemUsedK) / float64(s.SysMemK.MemTotal)

	s.LastProcCPUSample = s.CurProcCPUSample
	s.CurProcCPUSample = systemstat.GetProcCPUSample()
	if s.procCPUSampled {
		s.ProcCPUAvg = systemstat.GetProcCPUAverage(s.LastProcCPUSample, s.CurProcCPUSample, s.ProcUptime)
	}
	s.procCPUSampled = true
}


func agent(){
	coresToPegPtr = flag.Int64("coresToPeg", 0, "how many CPU cores would you like to artificially peg to 100% usage")

	flag.Parse()

	stats := NewStats()

	runtime.GOMAXPROCS(runtime.NumCPU())

	// WARNING: each call to burnCPU() will peg one core
	// of your machine to 100%
	// If you have code you'd like to drop in to this example,
	// just run "go yourCode()" instead of "go burnCPU()
	for i := *coresToPegPtr; i > 0; i-- {
		fmt.Println("pegging one more CPU core.")
		go burnCPU()
	}

	for {
		stats.GatherStats(true)
		stats.PrintStats()
		stats.setStats()

		// This next line lets out see the jsonified object
		// produced by systemstat
		//	printJson(stats, false)
		time.Sleep(3 * time.Second)
	}
}


//func main() {
//	// get command line flags
//	coresToPegPtr = flag.Int64("coresToPeg", 0, "how many CPU cores would you like to artificially peg to 100% usage")
//
//	flag.Parse()
//
//	// this will help us poll the OS to get system statistics
//	stats := NewStats()
//
//	runtime.GOMAXPROCS(runtime.NumCPU())
//
//	// WARNING: each call to burnCPU() will peg one core
//	// of your machine to 100%
//	// If you have code you'd like to drop in to this example,
//	// just run "go yourCode()" instead of "go burnCPU()
//	for i := *coresToPegPtr; i > 0; i-- {
//		fmt.Println("pegging one more CPU core.")
//		go burnCPU()
//	}
//
//	for {
//		stats.GatherStats(true)
//		stats.PrintStats()
//
//		// This next line lets out see the jsonified object
//		// produced by systemstat
//		//	printJson(stats, false)
//		time.Sleep(3 * time.Second)
//	}
//}

func printJson(s *stats, indent bool) {
	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println("error:", err)
	}
	dst := new(bytes.Buffer)
	if indent {
		json.Indent(dst, b, "", "   ")
	} else {
		dst.Write(b)
	}
	fmt.Println(dst.String())
	time.Sleep(time.Second * 3)
}

func burnCPU() {
	time.Sleep(4 * time.Second)
	for {
		b := 1.0
		c := 1.0
		d := 1.0
		for j := 1; j < 1000; j++ {
			b *= float64(j)
			for k := 1; k < 700000; k++ {

				c *= float64(k)
				d = (28 + b*b/3.23412) / math.Sqrt(c*c)
				c *= d
			}
			time.Sleep(500 * time.Nanosecond)
			runtime.Gosched()
		}
		time.Sleep(10 * time.Second)
	}
}
