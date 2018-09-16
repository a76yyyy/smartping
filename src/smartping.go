package main

import (
	"flag"
	"fmt"
	"github.com/smartping/smartping/src/funcs"
	"github.com/smartping/smartping/src/g"
	"github.com/smartping/smartping/src/http"
	"github.com/jakecoffman/cron"
	"os"
	"runtime"
	"sync"
)

// Init config
var Version = "0.6.0"

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	version := flag.Bool("v", false, "show version")
	flag.Parse()
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}
	g.ParseConfig(Version)

	c := cron.New()
	c.AddFunc("*/60 * * * * *", func() {
		var wg sync.WaitGroup
		for _, target := range g.Cfg.Targets {
			if target.Addr != g.Cfg.Ip {
				wg.Add(1)
				go funcs.StartPing(target, &wg)
			}
		}
		wg.Wait()
		go funcs.StartAlert()
		if g.Cfg.Mode == "cloud" {
			go funcs.StartCloudMonitor(1)
		}
	}, "ping")
	c.AddFunc("*/300 * * * * *", func() {
		go funcs.ClearBucket()
		go funcs.ClearPingLog()
	}, "mtc")
	c.Start()
	http.StartHttp()
}
