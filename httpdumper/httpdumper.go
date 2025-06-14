package httpdumper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
)

func getHandle(cfg *Config) (*pcap.Handle, error) {
	if cfg.PcapFile == "" && cfg.Device == "" {
		return nil, errors.New("you must specify either an interface with -i or a pcap file with -r")
	}
	if cfg.PcapFile != "" && cfg.Device != "" {
		return nil, errors.New("both -i and -r are specified. Reading from pcap file will take precedence")
	}

	// 优先文件处理
	if cfg.PcapFile != "" {
		log.Printf("Reading from pcap file: %s\n", cfg.PcapFile)
		return pcap.OpenOffline(cfg.PcapFile)
	}

	if cfg.Device == "" {
		devices, err := pcap.FindAllDevs()
		if err != nil || len(devices) == 0 {
			return nil, errors.New("could not find any network interfaces. Please specify one with -i")
		}
		cfg.Device = devices[0].Name
	}
	log.Printf("Starting capture on interface: %s\n", cfg.Device)
	return pcap.OpenLive(cfg.Device, int32(cfg.snapLen), cfg.PromiscuousMode, pcap.BlockForever)

}

func (hd *HttpDumper) processPackets(handle *pcap.Handle) {
	streamFactory := &httpStreamFactory{notifier: hd.n, Verbose: hd.cfg.Verbose}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)

	log.Println("Waiting for packets...")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

_out:
	for {
		select {
		case packet := <-packets:
			if packet == nil {
				log.Println("End of packet stream.")
				assembler.FlushAll()
				break _out
			}
			if packet.ErrorLayer() != nil {
				log.Println("Error decoding a packet:", packet.ErrorLayer().Error())
				continue
			}
			if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
				tcp, _ := tcpLayer.(*layers.TCP)
				assembler.Assemble(packet.NetworkLayer().NetworkFlow(), tcp)
				//assembler.AssembleWithContext(packet.NetworkLayer().NetworkFlow(), tcp, nil)
			}
		case <-ticker.C:
			flushed, closed := assembler.FlushOlderThan(time.Now().Add(-2 * time.Minute))
			if flushed > 0 {
				log.Printf("Flushed %d old streams, closed %d streams\n", flushed, closed)
			}
		case <-hd.ctx.Done():
			assembler.FlushAll()
			break _out
		}
	}

	streamFactory.wg.Wait()
	log.Println("done")
}

type HttpDumper struct {
	cfg    *Config
	n      Notifier
	ctx    context.Context
	cancel func()
}

func New(cfg *Config, n Notifier) *HttpDumper {
	return &HttpDumper{
		cfg: cfg,
		n:   n,
	}
}

func (hd *HttpDumper) Start(ctx context.Context) error {
	hd.ctx, hd.cancel = context.WithCancel(ctx)

	// 打开设备
	handle, err := getHandle(hd.cfg)
	if err != nil {
		return fmt.Errorf("error opening pcap handle: %v", err)
	}
	defer handle.Close()

	// 设置过滤器
	if err = handle.SetBPFFilter(hd.cfg.BPFFilter); err != nil {
		return fmt.Errorf("error setting BPF filter: %v", err)
	}
	log.Printf("Using BPF filter: %s\n", hd.cfg.BPFFilter)

	// 处理数据
	hd.processPackets(handle)

	return nil
}

func (hd *HttpDumper) Stop() {
	hd.cancel()
	log.Println("stop packets processing...")
}
