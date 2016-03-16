package main

import (
    "fmt"
    "os"
    "sort"
    "strings"
    "sync"
    "time"

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
var totalHitsCounter uint64

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
            Name:   `with-section, S`,
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
            Name:   `request-hits-alerts, A`,
            Usage:  `Show alerts when the total hits of requests per history window exceeds a configured threshold`,
        },
        cli.IntFlag{
            Name:   `requests-max-hits, R`,
            Usage:  `The maximum average hit count threshold`,
            Value:  DEFAULT_MAX_REQUESTS_PER_SEC,
        },
        cli.IntFlag{
            Name:   `request-hits-history, H`,
            Usage:  `How many observations to store (at per-second resoltion) when averaging the total hit count for alerting`,
            Value:  DEFAULT_REQUEST_RATE_HISTORY,
        },
        cli.BoolFlag{
            Name:   `no-color`,
            Usage:  `Disable colors in terminal output`,
        },
    }

    app.Action = func(c *cli.Context) {
        log.SetOutput(os.Stderr)

        switch c.String(`log-level`) {
        case `info`:
            log.SetLevel(log.InfoLevel)
        case `warn`:
            log.SetLevel(log.WarnLevel)
        case `error`:
            log.SetLevel(log.ErrorLevel)
        case `fatal`:
            log.SetLevel(log.FatalLevel)
        case `quiet`:
            log.SetLevel(log.PanicLevel)
        default:
            log.SetLevel(log.DebugLevel)
        }

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
                    totalHitsCounter += 1

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

    //  allocate ring buffer if we're monitoring average hit count
        if c.Bool(`request-hits-alerts`) {
            log.Debugf("Monitoring total hits (average over %d seconds should not exceed %d)", c.Int(`request-hits-history`), c.Int(`requests-max-hits`))
            totalReqHistory = NewRing(c.Int(`request-hits-history`))
        }

        fmt.Printf("section \tcount \tresponses \n")

        for {
        //  update and reset hits/sec counter
            UpdateHitCounter()

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


    sections := make([]*LogStatistic, 0)

    var topSection *LogStatistic

//  because we're working with a map that is accessed/modified across goroutines,
//  we grab a mutex to safely iterate over it without risk of it changing midway through
    mx.Lock()

    for _, stat := range sectionStats {
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
            if onlySections := c.StringSlice(`with-section`); len(onlySections) > 0 {
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

//  release mutex
    mx.Unlock()

    if topSection != nil {
        sections = []*LogStatistic{ topSection }
    }

//  only print rollups and reset counters every <interval> seconds
    if observations % c.Int(`interval`) == 0 || forced {
        log.Infof("Time: %s", time.Now().Format(time.RFC3339))

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

    if c.Bool(`request-hits-alerts`) {
        mx.Lock()

    //  if at least <request-hits-history> writes have occurred since the last clear, we can check the history
    //  for whether we should alert or not
        if totalReqHistory.WriteCount() >= totalReqHistory.Length() {
            var avgHits uint64

            // log.Debugf("  Checking: %+v", totalReqHistory.Data)

            for _, v := range totalReqHistory.Data {
                switch v.(type) {
                case uint64:
                    avgHits += v.(uint64)
                default:
                    log.Errorf("Unhandled history value type %T", v)
                }
            }

            avgHits = avgHits / uint64(totalReqHistory.Length())


        //  if the alert is in a triggered state, then we're checking to see if it has cleared
            if alertTriggered {
                if avgHits < uint64(c.Int(`requests-max-hits`)) {
                    log.Infof("Traffic has returned to normal levels - hits = %d at %s", avgHits, time.Now())
                    alertTriggered = false

                //  clear the history to force it to re-accumulate in order to trigger the alert again
                    totalReqHistory.Clear()
                }

        //  ...otherwise, we check to see if we should be firing the alert
            }else{
                if avgHits > uint64(c.Int(`requests-max-hits`)) {
                    log.Errorf("High traffic generated an alert - hits = %d, triggered at %s", avgHits, time.Now())
                    alertTriggered = true

                //  clear the history to force it to re-accumulate in order to clear the alert
                    totalReqHistory.Clear()
                }else{
                    log.Debugf("Traffic in bounds (average: %d hits)", avgHits)
                }
            }
        }else{
            log.Debugf("Populating: %d/%d", totalReqHistory.WriteCount(), totalReqHistory.Length())
        }

        mx.Unlock()
    }

//  break if we've reached a desired number of iterations
    if c.Int(`count`) > 0 && observations >= c.Int(`count`) {
        return
    }
}


// This function will push the current hit count into the ring buffer
// and then reset the count (synchronously)
//
func UpdateHitCounter() {
    mx.Lock()

    totalReqHistory.Push(totalHitsCounter)
    totalHitsCounter = 0

    mx.Unlock()
}