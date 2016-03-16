# logstat: a log file monitor
logstat is a utility designed to read [NCSA-formatted](https://en.wikipedia.org/wiki/Common_Log_Format) log files (such as used by the Apache HTTPD web server) and summarize their contents, either once or periodically on a specified interval.

## Getting Started

### Getting logstat
The most straightforward way to retrieve `logstat` is using the Golang `go get` subcommand:

```sh
go get github.com/ghetzel/logstat
```

### Building
If you want to build `logstat` from source, you can checkout the repository and build with the following commands:

```sh
git clone https://github.com/ghetzel/logstat.git
cd logstat
make all
```