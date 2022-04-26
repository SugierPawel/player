package main

import (
	"log"
	"net"
	"os"
	"strconv"

	. "github.com/SugierPawel/player/rpc/client"
	"github.com/SugierPawel/player/rpc/core"
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
		log.Println("dodanie kanału: AddRTPsource [IP in] [video port in] [audio port in] [nazwa]")
		log.Println("kasowanie kanału: DelRTPsource [IP in] [video port in]")
		os.Exit(3)
	} else {
		streamConfig = new(core.StreamConfig)
		streamConfig.Request = os.Args[1]
		switch streamConfig.Request {
		case "AddRTPsource":
			streamConfig.IPIn = net.ParseIP(os.Args[2]).To4().String()
			streamConfig.VideoPortIn, _ = strconv.Atoi(os.Args[3])
			streamConfig.AudioPortIn, _ = strconv.Atoi(os.Args[4])
			streamConfig.ChannelName = os.Args[5]
		case "DelRTPsource":
			streamConfig.IPIn = net.ParseIP(os.Args[2]).To4().String()
			streamConfig.VideoPortIn, _ = strconv.Atoi(os.Args[3])
		}
		log.Printf("streamConfig: %+v\n", streamConfig)
	}
	runClient()
}
