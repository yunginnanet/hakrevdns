package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"strings"
	"context"
	"time"
	"bufio"
	"sync"
	"fmt"
	"net"
	"os"
)

var Threads int = 8
var resolverList string = "./dnsResolvers.txt"
var targetList string = "targets.txt"
var Resolvers []string
var Protocol string = "udp"
var Timeout int = 3 // timeout int in seconds
var Port int = 53
var resolverIP string
var resList []string


func resolverLoad(path string) []string {
	log.Info().Msg("Loading IP addresses from " + path)

	f, err := os.Open(path)

	if err != nil {
		log.Debug().Msg(err.Error())
		os.Exit(1)
	}

	defer f.Close()
	scan := bufio.NewScanner(f)

	var l []string

	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.Count(line,".") == 3 {
			log.Debug().Msg("load from " + path + ": " + line)
			l = append(l, line)
		} else {
			log.Debug().Msg("invalid: " + line)
			os.Exit(1)
		}
	}

	log.Info().Int("count", len(l)).Msg("Done loading resolvers")

	return l
}


func targetLoad(path string) []string {
	log.Info().Msg("Loading target addresses from " + path)

	f, err := os.Open(path)

	if err != nil {
		log.Debug().Msg(err.Error())
		os.Exit(1)
	}

	defer f.Close()
	scan := bufio.NewScanner(f)

	var l []string

	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.Contains(line, "."){
			log.Debug().Msg("load from " + path + ": " + line)
			l = append(l, line)
		} else {
			log.Debug().Msg("invalid: " + line)
			os.Exit(1)
		}
	}

	log.Info().Int("count", len(l)).Msg("Done loading resolvers")

	return l
}





func ipLoad(path string) []string {
	log.Info().Msg("Loading IP addresses from " + path)

	f, err := os.Open(path)

	if err != nil {
		log.Debug().Msg(err.Error())
		os.Exit(1)
	}

	defer f.Close()
	scan := bufio.NewScanner(f)

	var l []string

	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.Count(line,".") == 3 {
			log.Debug().Msg("load from " + path + ": " + line)
			l = append(l, line)
		} else {
			log.Debug().Msg("invalid: " + line)
			os.Exit(1)
		}
	}

	log.Info().Int("count", len(l)).Msg("Done loading resolvers")

	return l
}


func randoIP(choice []string) string {
	len := len(choice)
	n := uint32(0)
	if len > 0 {
		n = getRandomUint32() % uint32(len)
	}
	return choice[n]
}

func getRandomUint32() uint32 {
	x := time.Now().UnixNano()
	return uint32((x >> 32) ^ x)
}

func init() {
	// init logger

	lf, err := os.OpenFile("./randrevdns.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatal().Err(err).Msg("Error opening log file!")
	}



	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr}
	multi := zerolog.MultiLevelWriter(consoleWriter, lf)
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
	// load resolvers from list
	Resolvers = ipLoad(resolverList)
}


func main() {
	work := make(chan string)
	go func() {
		f, err := os.Open(targetList)
		if err != nil {
			log.Debug().Err(err).Msg("fatal")
			os.Exit(1)
		}
		defer f.Close()
		ipList := bufio.NewScanner(f)
		for ipList.Scan() {
			target := ipList.Text()
			work <- target
		}
		close(work)
	}()

	wg := &sync.WaitGroup{}

	for i := 0; i < Threads; i++ {
		wg.Add(1)
		go doWork(work, wg)
	}
	wg.Wait()
}

func doWork(work chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	var r *net.Resolver

	tout := time.Duration(Timeout)*time.Second

	for ip := range work {

		resolverIP = randoIP(Resolvers)

		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: tout,
				}
				return d.DialContext(ctx, Protocol, fmt.Sprintf("%s:%d", resolverIP, Port))
			},
		}

		log.Debug().Str("ip",ip).Str("resolver",resolverIP).Msg("Resolving hostname with dialer")
		addr, err := r.LookupAddr(context.Background(), ip)
		if err != nil {
			log.Error().Err(err).Msg("fail")
			continue
		}

		for _, a := range addr {
			log.Debug().Str(ip, a).Msg("result")
		}
	}
}
