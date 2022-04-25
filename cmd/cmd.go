package main

import (
	"log"
	"net"
	"os"
	"strconv"

	. "github.com/SugierPawel/news/rpc/client"
	"github.com/SugierPawel/news/rpc/core"
)

var streamConfig *core.StreamConfig

func must(err error) {
	if err == nil {
		return
	}

	log.Panicln(err)
}

func runClient() {
	client := &Client{
		Port: core.RPC_PORT,
	}
	defer client.Close()
	must(client.Init())

	response, err := client.Execute(streamConfig)
	must(err)

	log.Println("response >>> ", response)
}

func main() {
	if len(os.Args) < 3 || os.Args[1] == "" {
		log.Println("dodanie kanału: AddRTPsource [IP in] [video port in] [audio port in] [broadcast IP out] [broadcast port out] [nazwa]")
		log.Println("kasowanie kanału: DelRTPsource [IP in] [video port in]")
		os.Exit(3)
	} else {
		streamConfig = new(core.StreamConfig)
		streamConfig.Request = os.Args[1]

		switch streamConfig.Request {
		case "AddRTPsource":

			IPIn := net.ParseIP(os.Args[2])
			BroadcastIP := net.ParseIP(os.Args[5])

			streamConfig.IPIn = IPIn.To4().String()
			streamConfig.VideoPortIn, _ = strconv.Atoi(os.Args[3])
			streamConfig.AudioPortIn, _ = strconv.Atoi(os.Args[4])
			streamConfig.BroadcastIP = BroadcastIP.To4().String()
			streamConfig.BroadcastPort, _ = strconv.ParseUint(os.Args[6], 10, 64)
			streamConfig.ChannelName = os.Args[7]

		case "DelRTPsource":

			IPIn := net.ParseIP(os.Args[2])
			streamConfig.IPIn = IPIn.To4().String()
			streamConfig.VideoPortIn, _ = strconv.Atoi(os.Args[3])
		}

		log.Printf("streamConfig: %+v\n", streamConfig)
	}
	runClient()
}
