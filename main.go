package main

import (
    "fmt"
    "os"
    "sort"
    "strings"
    "sync"
    "time"

    "./util"

    "github.com/codegangsta/cli"
    "github.com/fatih/color"
    log "github.com/Sirupsen/logrus"
)

const VERSION                        = `0.0.1`
const DEFAULT_TOP_INTERVAL           = 10
const DEFAULT_TOP_COUNT              = -1
const DEFAULT_MAX_REQUESTS_PER_SEC   = 100
const DEFAULT_REQUEST_RATE_HISTORY   = 120

var blue   = color.New(color.FgBlue).SprintFunc()
var green  = color.New(color.FgGreen).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var red    = color.New(color.FgRed).SprintFunc()

var mx                    = new(sync.Mutex)
var alertTriggered        = false
var totalReqHistory *Ring
var sectionStats          = make(map[string]*LogStatistic)
var streamFinished        = make(chan bool)
var observations          = 0

func main(){
    app                      := cli.NewApp()
    app.Name                  = `logstat`
    app.Version               = VERSION
    app.EnableBashCompletion  = false
    app.Flags = []cli.Flag{
        cli.StringFlag{
            Name:   `log-level, L`,
            Usage:  `Level of log output verbosity`,
            Value:  `info`,
            EnvVar: `LOGLEVEL`,
        },
        cli.BoolTFlag{
            Name:   `top, t`,
            Usage:  `Show top site sections`,
        },
        cli.StringSliceFlag{
            Name:   `only-sections, S`,
            Usage:  `When displaying multiple sections (--top=false), choose which ones to show`,
        },
        cli.IntFlag{
            Name:   `count, c`,
            Usage:  `The maximum number of iterations to output in "top" mode`,
            Value:  DEFAULT_TOP_COUNT,
        },
        cli.IntFlag{
            Name:   `interval, i`,
            Usage:  `Interval (in seconds) that the top output window will summarize results for`,
            Value:  DEFAULT_TOP_INTERVAL,
        },
        cli.BoolTFlag{
            Name:   `request-rate-alerts, A`,
            Usage:  `Show alerts when the total rate of requests seen exceeds a configured threshold`,
        },
        cli.IntFlag{
            Name:   `requests-max-rate, R`,
            Usage:  `The maximum average requests/sec threshold`,
            Value:  DEFAULT_MAX_REQUESTS_PER_SEC,
        },
        cli.IntFlag{
            Name:   `request-rate-history, H`,
            Usage:  `How many observations to store (at per-second resoltion) when averaging the request rate for alerting`,
            Value:  DEFAULT_REQUEST_RATE_HISTORY,
        },
        cli.BoolFlag{
            Name:   `no-color`,
            Usage:  `Disable colors in terminal output`,
        },
    }

    app.Action = func(c *cli.Context) {
        util.ParseLogLevel(c.String(`log-level`))

        if c.Bool(`no-color`) {
            color.NoColor = true

            log.SetFormatter(&log.TextFormatter{
                DisableColors: true,
            })
        }else{
            log.SetFormatter(&log.TextFormatter{
                ForceColors: true,
            })
        }

        log.Debugf("Starting %s %s", c.App.Name, c.App.Version)

        go func(){
            err := ParseStream(os.Stdin, func(logLine NcsaLog, err error){
                if err == nil {
                    mx.Lock()

                    parts := strings.Split(logLine.Path, `/`)

                //  this is where statistics are appended for each log line received
                    if len(parts) > 1 {
                        sectionName := strings.Split(parts[1], `?`)[0]

                        stat, ok := sectionStats[sectionName]


                        if !ok {
                            stat = NewLogStatistic(sectionName)
                            sectionStats[sectionName] = stat
                        }

                        stat.Count += 1
                        stat.Sizes = append(stat.Sizes, logLine.Size)
                        stat.Logs  = append(stat.Logs, &logLine)
                    }

                    mx.Unlock()
                }else{
                    log.Errorf("%v", err)
                }
            })

            streamFinished <- true

            if err != nil {
                log.Fatalf("Failed to parse log stream: %v", err)
            }
        }()

    //  allocate ring buffer if we're monitoring total request rate
        if c.Bool(`request-rate-alerts`) {
            log.Debugf("Monitoring total request rate (average over %d seconds should not exceed %d req/sec)", c.Int(`request-rate-history`), c.Int(`requests-max-rate`))
            totalReqHistory = NewRing(c.Int(`request-rate-history`))
        }

        fmt.Printf("section \tcount \tresponses \n")

        for {
            log.Infof("Time: %s", time.Now().Format(time.RFC3339))
            select {
            case <-streamFinished:
                ProcessLogs(c, true)
                return
            case <-time.After(time.Second):
                ProcessLogs(c, false)
            }
        }
    }

    app.Run(os.Args)
}

func ProcessLogs(c *cli.Context, forced bool) {
    observations += 1

    mx.Lock()


    sections := make([]*LogStatistic, 0)

    var topSection *LogStatistic
    var totalReqs uint64

    for _, stat := range sectionStats {

        totalReqs += stat.Count

    //  if we're in "top" mode, accumulate logs on an interval and summarize them
        if c.Bool(`top`) {
            if topSection == nil {
                topSection = stat
            }

            if stat.Count > topSection.Count {
                topSection = stat
            }

    //  ...otherwise, we're processing all sections that we have stats for
        }else{
            if onlySections := c.StringSlice(`only-sections`); len(onlySections) > 0 {
                for _, name := range onlySections {
                    if stat.Key == name {
                        sections = append(sections, stat)
                        break
                    }
                }

            }else{
                sections = append(sections, stat)
            }
        }
    }

    if topSection != nil {
        sections = []*LogStatistic{ topSection }
    }


    mx.Unlock()

//  push this iteration's total requests if we're monitoring total request rate
    if c.Bool(`request-rate-alerts`) {
        totalReqHistory.Push(totalReqs)
    }

//  only print rollups and reset counters every <interval> seconds
    if observations % c.Int(`interval`) == 0 || forced {
        for _, section := range sections {
            if section != nil {
                fmt.Printf("%s \t%d \t", section.Key, section.Count)

                families := make([]string, 0)

                for status, count := range section.GroupByStatusFamily() {
                    if count > 0 {
                        families = append(families, fmt.Sprintf("%s=%d", status, count))
                    }
                }

                sort.Strings(families)

                for _, fam := range families {
                    switch fam[0] {
                    case '1':
                    case '2':
                        fam = green(fam)
                    case '4':
                        fam = yellow(fam)
                    case '5':
                        fam = red(fam)
                    default:
                        fam = blue(fam)
                    }

                    fmt.Printf("%s ", fam)
                }

                fmt.Printf("\n")
            }else{
                fmt.Printf("%s \t%d\n", `-`, 0)
            }
        }

        mx.Lock()
        sectionStats = make(map[string]*LogStatistic)
        mx.Unlock()

    }

    if c.Bool(`request-rate-alerts`) {
    //  if at least <request-rate-history> writes have occurred since the last clear, we can check the history
    //  for whether we should alert or not
        if totalReqHistory.WriteCount() >= totalReqHistory.Length() {
            var avgRate uint64

            for _, v := range totalReqHistory.Data {
                switch v.(type) {
                case uint64:
                    avgRate += v.(uint64)
                }
            }

            avgRate = avgRate / uint64(totalReqHistory.Length())


        //  if the alert is in a triggered state, then we're checking to see if it has cleared
            if alertTriggered {
                if avgRate < uint64(c.Int(`requests-max-rate`)) {
                    log.Infof("Traffic rate has returned to normal levels - hits = %d req/sec at %s", avgRate, time.Now())
                    alertTriggered = false
                }

        //  ...otherwise, we check to see if we should be firing the alert
            }else{
                if avgRate > uint64(c.Int(`requests-max-rate`)) {
                    log.Errorf("High traffic generated an alert - hits = %d req/sec, triggered at %s", avgRate, time.Now())
                    alertTriggered = true

                //  clear the history to force it to re-accumulate in order to clear the alert
                    totalReqHistory.Clear()
                }else{
                    log.Debugf("Rate in bounds (%d req/sec)", avgRate)
                }
            }
        }else{
            log.Debugf("Waiting for rate history to populate (have %d, need >= %d)", totalReqHistory.WriteCount(), totalReqHistory.Length())
        }
    }

//  break if we've reached a desired number of iterations
    if c.Int(`count`) > 0 && observations >= c.Int(`count`) {
        return
    }
}