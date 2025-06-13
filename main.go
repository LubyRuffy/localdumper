package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/LubyRuffy/localdumper/httpdumper"
	"github.com/google/gopacket"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func parseConfig() *httpdumper.Config {
	var cfg httpdumper.Config
	flag.StringVar(&cfg.Device, "i", "", "Network interface to capture packets from. (e.g., en0, eth0)")
	flag.StringVar(&cfg.PcapFile, "r", "", "Pcap file to read packets from.")
	flag.StringVar(&cfg.BPFFilter, "f", "tcp", "BPF filter for capturing packets. Use 'tcp' for all TCP traffic.")
	//flag.IntVar(&cfg.SnapLen, "s", -1, "SnapLen for pcap packet capture.")
	flag.BoolVar(&cfg.PromiscuousMode, "p", false, "Set interface to promiscuous mode.")
	flag.Parse()
	return &cfg
}

type Notifier struct {
}

func (n *Notifier) OnRequest(req *httpdumper.Request) {
	fmt.Println(strings.Repeat(">", 58))
	fmt.Printf(">>> HTTP Request (ID: %s): %s:%s -> %s:%s\n",
		req.ID, req.Net.Src(), req.Transport.Src(), req.Net.Dst(), req.Transport.Dst())
	fmt.Printf("%s %s %s\n", req.Method, req.URL, req.Proto)
	fmt.Printf("Host: %s\n", req.Host)
	for key, values := range req.Header {
		fmt.Printf("%s: %s\n", key, strings.Join(values, ", "))
	}
	if len(req.Body) > 0 {
		fmt.Printf("\n%s\n", req.Body)
	}
	fmt.Println(strings.Repeat(">", 58))
}

func (n *Notifier) OnResponse(resp *httpdumper.Response) {
	fmt.Println(strings.Repeat("<", 58))
	id := ""
	if resp.Request != nil {
		id = resp.Request.ID
	}
	fmt.Printf("<<< HTTP Response (ID: %s): %s:%s <- %s:%s\n",
		id, resp.Net.Dst(), resp.Transport.Dst(), resp.Net.Src(), resp.Transport.Src())
	fmt.Printf("    %s %s\n", resp.Proto, resp.Status)
	for key, values := range resp.Header {
		fmt.Printf("    %s: %s\n", key, strings.Join(values, ", "))
	}
	if len(resp.Body) > 0 {
		fmt.Printf("\n%s\n", resp.Body)
	}
	fmt.Println(strings.Repeat("<", 58))
}

func (n *Notifier) OnTcpSession(id string, net, transport gopacket.Flow) {
	fmt.Printf("New TCP session: %s\n", id)
}

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	hd := httpdumper.New(parseConfig(), &Notifier{})
	doneChan := make(chan struct{}, 1)
	go func() {
		defer close(doneChan)
		if err := hd.Start(context.Background()); err != nil {
			panic(err)
		}
	}()

	<-signalChan
	fmt.Println("\nReceived interrupt, shutting down...")
	hd.Stop()
	<-doneChan
}
