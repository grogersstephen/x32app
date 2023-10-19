package main

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (f *fader) levelMessage() string {
	msg := fmt.Sprintf("%s %d", f.name, f.channel)
	if f.level < 0 {
		return fmt.Sprintf("%s : ??", msg)
	}
	return fmt.Sprintf("%s : %.2f", msg, f.level)
}

func isValidIP(ip string) bool {
	// True:
	//     If a valid ip address as defined by net.ParseIP
	//     If a port number with a leading colon ":10023"
	//     If a valid ip followed by a colon followed by a valid port number "192.168.1.1:10023"
	var port string
	ip = strings.TrimSpace(ip)
	// If it's an empty string, return false
	if ip == "" {
		return false
	}
	// If there's a colon, we're looking for a port number
	containsPort := strings.Count(ip, ":") == 1
	if containsPort {
		// the IP is the substring before the colon
		// the port is the substring after the colon
		ipPort := strings.Split(ip, ":")
		ip = ipPort[0]
		port = ipPort[1]
	}
	// if the ip is defined, check that we can parse it using net.ParseIP()
	if ip != "" {
		if net.ParseIP(ip) == nil {
			return false
		}
	}
	// if the port is defined, check that we can convert it to an int using strconv.Atoi()
	if port != "" {
		portI, err := strconv.Atoi(port)
		if err != nil {
			return false
		}
		if portI < 0 || portI > 65535 {
			return false
		}
	}
	return true
}

func getChannelIDPath(ch int) string {
	// Return prefix of an osc message corresponding to given ChannelID
	//     This is not a complete osc message recognizable by the X32
	switch {
	case ch < 0:
		return ""
	case ch < 32: // channel
		return fmt.Sprintf("/ch/%02d", ch+1)
	case ch < 40: // aux
		return fmt.Sprintf("/auxin/%02d", ch-31)
	case ch < 48: // fx
		return fmt.Sprintf("/fxrtn/%02d", ch-39)
	case ch < 64: // bus send
		return fmt.Sprintf("/bus/%02d", ch-47)
	case ch < 70: // matrix
		return fmt.Sprintf("/mtx/%02d", ch-63)
	case ch == 70: // stereo main
		return "/main/st"
	case ch == 71: // mono main
		return "/main/m"
	case ch < 80: // dca
		return fmt.Sprintf("/dca/%d", ch-71)
	default:
		return ""
	}
}

func getNamePath(ch int) string {
	path := getChannelIDPath(ch)
	return filepath.Join(path, "config/name")
}

func getFaderPath(ch int) string {
	path := getChannelIDPath(ch)
	if path == "" {
		return path
	}
	// Return osc message corresponding to the fader of the given channelID
	if ch > 71 && ch < 80 { // dca
		//dca syntax differs: "/dca/3/fader"
		return filepath.Join(path, "fader")
	}
	// e.g. channel 1: /ch/01/mix/fader
	return filepath.Join(path, "mix/fader")
}

func getDist(x, y int) int {
	// Returns the absolute distance between two integers
	if x < y {
		return y - x
	}
	return x - y
}

func loadingAnimation(console chan string, doneSignal chan bool) {
	// Start loading animation
	go func() {
		for i := 1; i > 0; i++ {
			n := i % 7
			cs := fmt.Sprintf("%s%s", "connecting", strings.Repeat(".", n))
			time.Sleep(250 * time.Millisecond)
			select {
			case <-doneSignal:
				return
			default:
			}
			console <- "clr"
			console <- cs
		}
	}()
}
