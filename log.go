package main

import (
    "bufio"
    "fmt"
    "io"
    "regexp"
    "strconv"
    "strings"
    "time"
)

const NCSA_RX               = `^(?P<host>(?:\d{1,3}[\.]){3}\d{1,3}) (?P<id>\S+) (?P<user>\S+) \[(?P<timestamp>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>[^"]+)" (?P<status>\d+) (?P<size>\d+) ?(?P<rest>.*)`
const NCSA_TIMESTAMP_LAYOUT = `2/Jan/2006:15:04:05 -0700`

type LogStatistic struct {
    Key     string
    Count   uint64
    Sizes   []uint64
    Logs    []*NcsaLog
}

func NewLogStatistic(key string) *LogStatistic {
    return &LogStatistic{
        Key:     key,
        Count:   0,
        Sizes:   make([]uint64, 0),
        Logs:    make([]*NcsaLog, 0),
    }
}

func (self *LogStatistic) AvarageSize() float64 {
    var sz uint64

    for _, s := range self.Sizes {
        sz += s
    }

    return float64(sz / uint64(len(self.Sizes)))
}


func (self *LogStatistic) GroupByStatusFamily() map[string]uint64 {
    statuses := map[string]uint64{
        `1xx`: 0,
        `2xx`: 0,
        `3xx`: 0,
        `4xx`: 0,
        `5xx`: 0,
        `???`: 0,
    }

    for _, logLine := range self.Logs {
        if logLine.StatusCode < 200 {
            if v, ok := statuses[`1xx`]; ok {
                statuses[`1xx`] = (v + 1)
            }
        }else if logLine.StatusCode < 300 {
            if v, ok := statuses[`2xx`]; ok {
                statuses[`2xx`] = (v + 1)
            }
        }else if logLine.StatusCode < 400 {
            if v, ok := statuses[`3xx`]; ok {
                statuses[`3xx`] = (v + 1)
            }
        }else if logLine.StatusCode < 500 {
            if v, ok := statuses[`4xx`]; ok {
                statuses[`4xx`] = (v + 1)
            }
        }else if logLine.StatusCode < 600 {
            if v, ok := statuses[`5xx`]; ok {
                statuses[`5xx`] = (v + 1)
            }
        }else{
            if v, ok := statuses[`???`]; ok {
                statuses[`???`] = (v + 1)
            }
        }
    }

    return statuses
}

type LogCallback func(NcsaLog, error)

type NcsaLog struct {
    Host       string
    Identity   string
    UserId     string
    Timestamp  time.Time
    Method     string
    Path       string
    Protocol   string
    StatusCode uint
    Size       uint64
    Rest       string
}

func ParseStream(input io.Reader, cb LogCallback) error {
    lineScanner := bufio.NewScanner(input)

    for lineScanner.Scan() {
        line := lineScanner.Text()
        logEntry := NcsaLog{}

        err := logEntry.Parse(line)

    //  call the callback for this log line
    //  NOTE: I could have used a channel here, but I decided to err on the side of caution
    //        between "demonstrate idiomatic use" and "being too clever"
    //
        cb(logEntry, err)
    }

    return lineScanner.Err()
}

func (self *NcsaLog) Parse(line string) error {
    if rx, err := regexp.Compile(NCSA_RX); err == nil {
        if match := rx.FindStringSubmatch(line); match != nil {
            for i, field := range rx.SubexpNames() {
                if match[i] == `-` {
                    continue
                }

                switch field {
                case `host`:
                    self.Host = match[i]
                case `id`:
                    self.Identity = match[i]
                case `user`:
                    self.UserId = match[i]
                case `timestamp`:
                    if tm, err := time.Parse(NCSA_TIMESTAMP_LAYOUT, match[i]); err == nil {
                        self.Timestamp = tm
                    }else{
                        return err
                    }
                case `method`:
                    self.Method = strings.ToUpper(match[i])
                case `path`:
                    self.Path = match[i]
                case `protocol`:
                    self.Protocol = match[i]
                case `status`:
                    if v, err := strconv.ParseUint(match[i], 10, 16); err == nil {
                        self.StatusCode = uint(v)
                    }else{
                        return err
                    }
                case `size`:
                    if v, err := strconv.ParseUint(match[i], 10, 64); err == nil {
                        self.Size = v
                    }else{
                        return err
                    }
                case `rest`:
                    if rest := strings.TrimSpace(match[i]); len(rest) > 0 {
                        self.Rest = rest
                    }
                }
            }

        }else{
            return fmt.Errorf("Input did not match parse format: '%s'", line)
        }
    }else{
        return err
    }

    return nil
}