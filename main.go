package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"

	"github.com/SugierPawel/player/ini"
	"github.com/SugierPawel/player/rpc/core"
	"github.com/SugierPawel/player/wss"
)

const (
	updWriteSleepTime = 100000
	sleepTime         = time.Millisecond * 100
	serverAddress     = "172.26.9.100"          //"172.26.9.100"
	webSocketAddr     = serverAddress + ":2000" //"172.26.9.100:2000" "127.0.0.1:2000"
)

var wssHub *wss.Hub
var sourceMutex sync.Mutex
var updSourceMap map[string]*updSource
var ReceiversWebrtcMap map[string]*SourceToWebrtcConfig
var remoteP2PQueueMap map[string]*remoteP2PQueueConfig
var codecMap map[string]Codecs

type updSource struct {
	wg           *sync.WaitGroup
	ctx          context.Context
	rtcpConn     *net.UDPConn
	rtpConn      *net.UDPConn
	cancel       context.CancelFunc
	ssrcMap      map[string]string
	ssrcMutex    sync.Mutex
	pktsChanMap  map[string]chan *rtp.Packet
	depacketizer map[string]rtp.Depacketizer
	sampleBuffer map[string]*samplebuilder.SampleBuilder
	tracks       map[string]*webrtc.TrackLocalStaticSample
}
type iceConfig struct {
	client *wss.Client
	init   webrtc.ICECandidateInit
}
type offerConfig struct {
	client  *wss.Client
	offer   string
	channel string
}
type remoteP2PQueueConfig struct {
	ice   chan iceConfig
	offer chan offerConfig
}
type SourceToWebrtcConfig struct {
	access               sync.Mutex
	mediaEngine          *webrtc.MediaEngine
	interceptor          *interceptor.Registry
	settingEngine        *webrtc.SettingEngine
	api                  *webrtc.API
	peerConnection       *webrtc.PeerConnection
	answerPeerConnection *webrtc.PeerConnection
	remoteDescription    *webrtc.SessionDescription
	actualChannel        string
	ssrcMap              map[string]string
	receiverExitChan     chan bool
}
type JsMessage struct {
	Request string `json:"request"`
	Data    string `json:"data"`
	Channel string `json:"channel"`
}
type Codecs struct {
	MimeType      string
	SampleRate    int
	PacketMaxLate int
	dep           rtp.Depacketizer
}

var webrtcConfiguration = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs:       []string{"turn:" + serverAddress + ":5900"},
			Username:   "turnserver",
			Credential: "turnserver",
		},
		{
			URLs: []string{"stun:" + serverAddress + ":5900"},
		},
	},
}

func check(FunctionName string, sn string, err error) {
	if err != nil {
		log.Printf("ERROR - FunctionName: %s, sn: %s >> %s", FunctionName, sn, err)
	}
}

func AddRTPsource(sc *core.StreamConfig) {
	log.Printf("AddRTPsource: %+v\n", sc)
	sn := sc.StreamName
	sourceMutex.Lock()
	updSourceMap[sn] = new(updSource)
	updSourceMap[sn].wg = &sync.WaitGroup{}
	updSourceMap[sn].ctx, updSourceMap[sn].cancel = context.WithCancel(context.Background())
	updSourceMap[sn].ssrcMap = make(map[string]string)
	updSourceMap[sn].pktsChanMap = make(map[string]chan *rtp.Packet)
	updSourceMap[sn].pktsChanMap["audio"] = make(chan *rtp.Packet)
	updSourceMap[sn].pktsChanMap["video"] = make(chan *rtp.Packet)
	updSourceMap[sn].depacketizer = make(map[string]rtp.Depacketizer)
	updSourceMap[sn].sampleBuffer = make(map[string]*samplebuilder.SampleBuilder)
	updSourceMap[sn].tracks = make(map[string]*webrtc.TrackLocalStaticSample)
	for kind := range codecMap {
		updSourceMap[sn].depacketizer[kind] = codecMap[kind].dep
		updSourceMap[sn].sampleBuffer[kind] = samplebuilder.New(
			uint16(codecMap[kind].PacketMaxLate),
			updSourceMap[sn].depacketizer[kind],
			uint32(codecMap[kind].SampleRate))
		var err error
		updSourceMap[sn].tracks[kind], err = webrtc.NewTrackLocalStaticSample(
			webrtc.RTPCodecCapability{MimeType: codecMap[kind].MimeType, ClockRate: uint32(codecMap[kind].SampleRate)},
			kind,
			sc.ChannelName)
		if err != nil {
			log.Printf("InitRtpReader - NewTrackLocalStaticSample, kind: %s, error: %s", kind, err)
		}
	}

	//go updSourceMap[sn].InitRtcpReader(sc)
	go updSourceMap[sn].InitRtpReader(sc)
	go updSourceMap[sn].InitRtpWriter(sc, "video")
	go updSourceMap[sn].InitRtpWriter(sc, "audio")

	jsonStr, _ := json.Marshal(sc)
	data, _ := json.Marshal(&JsMessage{
		Request: "addChannel",
		Data:    base64.URLEncoding.EncodeToString([]byte(jsonStr)),
	})
	wssHub.Broadcast <- data
	sourceMutex.Unlock()
}
func DelRTPsource(sc *core.StreamConfig) {
	log.Printf("DelRTPsource: %+v\n", sc)
	sn := sc.StreamName

	jsonStr, _ := json.Marshal(sc)
	data, _ := json.Marshal(&JsMessage{
		Request: "delChannel",
		Data:    base64.URLEncoding.EncodeToString([]byte(jsonStr)),
	})
	wssHub.Broadcast <- data

	updSourceMap[sn].wg.Add(3)
	updSourceMap[sn].cancel()
	updSourceMap[sn].wg.Wait()

	delete(updSourceMap, sn)
}

func (l *updSource) InitRtcpReader(sc *core.StreamConfig) {
	defer func() {
		l.wg.Done()
	}()
	sn := sc.StreamName
	var err error
	IPIn := ini.SCMap[sn].IPIn
	var port = ini.SCMap[sn].PortIn
	rtcpPort := port + 1
	//rtcpVideoFBPort := port + 2
	//rtcpAudiooFBPort := port + 3

	l.rtcpConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(IPIn), Port: rtcpPort})
	if err != nil {
		log.Printf("InitRtcp, sn: %s, IPIn: %s, port: %d, err: %s", sn, IPIn, rtcpPort, err)
		return
	}
	for {
		select {
		case <-l.ctx.Done():
			log.Printf("InitRtcp, sn: %s, ctx.Done()", sn)
			l.rtcpConn.Close()
			return
		default:
			p := make([]byte, 1500)
			rtcpN, _, err := l.rtcpConn.ReadFrom(p)
			if err != nil {
				log.Printf("InitRtcp, sn: %s, ReadFrom error: %s", sn, err)
				break
			}
			sr := &rtcp.SenderReport{}
			sr.Unmarshal(p[:rtcpN])

			l.ssrcMutex.Lock()
			var kind string
			if fmt.Sprint(sr.SSRC) == l.ssrcMap["video"] {
				kind = "video"
			} else if fmt.Sprint(sr.SSRC) == l.ssrcMap["audio"] {
				kind = "audio"
			}
			l.ssrcMutex.Unlock()

			/*for n, packet := range packets {
				log.Printf("InitRtcp << sn: %s, n: %d, SSRC: %d", sn, n, packet.DestinationSSRC())
				//packets[n] = packet.DestinationSSRC()
			}*/

			for receiverSN, config := range ReceiversWebrtcMap {
				if config.actualChannel == sn {
					intVar, _ := strconv.Atoi(ReceiversWebrtcMap[receiverSN].ssrcMap[kind])
					var ssrc uint32 = uint32(intVar)
					//preSSRC := sr.SSRC
					sr.SSRC = ssrc

					//errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})

					err = config.peerConnection.WriteRTCP([]rtcp.Packet{sr})
					if err != nil {
						log.Printf(">> InitRtcp >> sn: %s, WriteRTCP error: %s", sn, err)
					}
					//log.Printf(">> InitRtcp >> sn: %s, kind: %s, preSSRC: %d, DestinationSSRC: %d", sn, kind, preSSRC, sr.DestinationSSRC())
					/*
						for _, sender := range config.peerConnection.GetSenders() {
							senderKind := sender.Track().Kind().String()
							if senderKind != kind {
								continue
							}
							intVar, _ := strconv.Atoi(ReceiversWebrtcMap[receiverSN].ssrcMap[senderKind])
							var ssrc uint32 = uint32(intVar)
							preSSRC := sr.SSRC
							sr.SSRC = ssrc

							writed, err := sender.Transport().WriteRTCP([]rtcp.Packet{sr})
							if err != nil {
								log.Printf(">> InitRtcp >> sn: %s, WriteRTCP error: %s", sn, err)
							}
							log.Printf(">> InitRtcp >> sn: %s, senderKind: %s, preSSRC: %d, DestinationSSRC: %d, writed: %d", sn, senderKind, preSSRC, sr.DestinationSSRC(), writed)
						}

						for _, sender := range config.peerConnection.GetSenders() {
							switch sender.Track().Kind().String() {
							case "video":

							case "audio":

							}

							//log.Printf(", !!!!!!!!!!!!!!! %s ", TracksMap[sn].Direction["Broadcast"].kind[sender.Track().Kind().String()])
							//TracksMap[oc.channel].Direction["Broadcast"].ssrcMap["video"]

							s, err := sender.Transport().WriteRTCP(pas)
							if err != nil {
								log.Printf("InitRtcp << błąd wysłania do rec: %s, err: %s", rec, err)
							} else {
								log.Printf("InitRtcp << wysłano: %d, do rec: %s, StreamID: %s", s, rec, sender.Track().StreamID())
							}
						}*/
					//err = config.peerConnection.WriteRTCP(packets)
				}
			}

		}
	}
}
func (l *updSource) InitRtpReader(sc *core.StreamConfig) {
	defer func() {
		l.wg.Done()
	}()
	sn := sc.StreamName
	var err error
	IPIn := ini.SCMap[sn].IPIn
	var port = ini.SCMap[sn].PortIn

	//l.rtpConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(IPIn), Port: port})
	l.rtpConn, err = net.ListenMulticastUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(IPIn), Port: port})
	if err != nil {
		log.Printf("InitRtpReader, sn: %s, IPIn: %s, port: %d, err: %s", sn, IPIn, port, err)
		return
	}
	var kind string
	for {
		select {
		case <-l.ctx.Done():
			log.Printf("rtpConn, sn: %s, ctx.Done()", sn)
			l.rtpConn.Close()
			return
		default:
			packet := make([]byte, 1200)
			n, _, err := l.rtpConn.ReadFrom(packet)
			if err != nil {
				log.Printf("InitRtpReader, sn: %s, ReadFrom error: %s", sn, err)
				break
			}
			rtpPacket := &rtp.Packet{}
			if err = rtpPacket.Unmarshal(packet[:n]); err != nil {
				log.Printf("InitRtpReader, sn: %s, rtpPacket.Unmarshal error: %s", sn, err)
				break
			}
			kind = "na"
			switch rtpPacket.Header.PayloadType {
			case 33:
			case 96:
				kind = "video"
			case 97:
				kind = "audio"
				//if sn == "224.11.11.1:1111" {
				//	log.Printf("InitRtpReader, sn: %s, kind: %s, n: %d, ts: %d, sn: %d", sn, kind, n, rtpPacket.Header.Timestamp, rtpPacket.Header.SequenceNumber)
				//}
			}
			/*if n > 30 && n < 500 {
				kind = "audio"
			} else if n > 500 {
				kind = "video"
			}*/
			//log.Printf("InitRtpReader, sn: %s, kind: %s, n: %d, payload: %d", sn, kind, n, rtpPacket.Header.PayloadType)

			if kind == "na" {
				break
			}
			l.pktsChanMap[kind] <- rtpPacket
			if _, ok := l.ssrcMap[kind]; !ok {
				l.ssrcMutex.Lock()
				l.ssrcMap[kind] = fmt.Sprint(rtpPacket.SSRC)
				l.ssrcMutex.Unlock()
			}
		}
	}
}
func (l *updSource) InitRtpWriter(sc *core.StreamConfig, kind string) {
	defer func() {
		l.wg.Done()
	}()
	sn := sc.StreamName
	for {
		select {
		case <-l.ctx.Done():
			log.Printf("rtpConn, sn: %s, ctx.Done()", sn)
			return
		case rtpPacket := <-l.pktsChanMap[kind]:
			l.sampleBuffer[kind].Push(rtpPacket)
			for {
				sample := l.sampleBuffer[kind].Pop()
				if sample == nil {
					//log.Printf("InitRtp << nie gotowy...., kind: %s", kind)
					break
				}
				//WriteSample!!!, sn: %s, kind: %s, ts: %d, dropped: %d", sn, kind, sample.PacketTimestamp, sample.PrevDroppedPackets)
				if sample.PrevDroppedPackets > 0 {
					log.Printf("InitRtp >> WriteSample, sn: %s, kind: %s, ts: %d, dropped: %d", sn, kind, sample.PacketTimestamp, sample.PrevDroppedPackets)
				}
				if err := l.tracks[kind].WriteSample(*sample); err != nil {
					log.Printf("InitRtp, kind: %s, sn: %s, WriteSample error: %s", kind, sn, err)
				}
			}
			//case <-time.After(updWriteSleepTime):
		}
	}
}

func StartWebSocketServer() {
	log.Println("StartWebSocketServer")
	wssHub = wss.NewHub()
	go wssHub.Run()
	http.HandleFunc("/signal", func(w http.ResponseWriter, r *http.Request) {
		wss.WebSocketAccept(wssHub, w, r)
	})
	go func() {
		err := http.ListenAndServe(webSocketAddr, nil)
		if err != nil {
			log.Printf("StartWebSocketServer err: %s", err)
		}
	}()
	for {
		select {
		case cm := <-wssHub.UnregisterReceiver:
			unRegisterReceiver(cm.Client)
		case cm := <-wssHub.RegisterReceiver:
			go registerReceiver(cm.Client)
		case cm := <-wssHub.Receiver:
			var jsMsg JsMessage
			json.Unmarshal([]byte(cm.Message), &jsMsg)
			uDec, _ := base64.URLEncoding.DecodeString(jsMsg.Data)
			jsMsg.Data = string(uDec)
			//log.Printf("Receiver: %v, %s, %s", cm.Client.Conn.RemoteAddr(), jsMsg.Request, jsMsg.Data)
			switch jsMsg.Request {
			case "offer":
				addRemoteOffer(cm.Client, jsMsg)
			case "ice":
				addRemoteIce(cm.Client, jsMsg)
			case "channel":
				//changeChannel(cm.Client, jsMsg.Data)
			}
		case <-time.After(sleepTime):
		}
	}
}
func addRemoteIce(client *wss.Client, jsMsg JsMessage) {
	var sn = client.Conn.RemoteAddr().String()
	ic := new(iceConfig)
	ic.client = client
	json.Unmarshal([]byte(jsMsg.Data), &ic.init)
	remoteP2PQueueMap[sn].ice <- *ic
}
func addRemoteOffer(client *wss.Client, jsMsg JsMessage) {
	var sn = client.Conn.RemoteAddr().String()
	oc := new(offerConfig)
	oc.client = client
	oc.offer = jsMsg.Data
	oc.channel = jsMsg.Channel
	remoteP2PQueueMap[sn].offer <- *oc
}
func changeChannel(client *wss.Client, channel string) {
	var fName string = "changeChannel"
	var sn = client.Conn.RemoteAddr().String()
	if _, ok := updSourceMap[channel]; !ok {
		log.Printf(" << CHANGE CHANNEL << klient: %s, brak aktywnego źródła rtp:// %s", sn, channel)
		return
	}
	if _, ok := ReceiversWebrtcMap[sn]; !ok {
		log.Printf(" << CHANGE CHANNEL << klient: %s nie ma aktywnego połączenia WebRTC ('js':NEWSWEBRTC.rtcpConnect()), channel: %s", sn, channel)
		return
	}
	if ReceiversWebrtcMap[sn].actualChannel == channel {
		log.Printf(" << CHANGE CHANNEL << klient: %s, kanał: %s jest już aktywny", sn, channel)
		return
	} else if ReceiversWebrtcMap[sn].actualChannel != "" {
		log.Printf(" << REPLACE CHANNEL << klient: %s, zamieniam: %s / %s", sn, ReceiversWebrtcMap[sn].actualChannel, channel)
	} else if ReceiversWebrtcMap[sn].actualChannel == "" {
		log.Printf(" << ASIGN CHANNEL << klient: %s, kanał: %s", sn, channel)
		for _, track := range updSourceMap[channel].tracks {
			_, err := ReceiversWebrtcMap[sn].peerConnection.AddTrack(track)
			check(fName, sn, err)
			/*go func() {
				rtcpBuf := make([]byte, 1500)
				for {
					n, a, rtcpErr := sender.Read(rtcpBuf)
					if rtcpErr != nil {
						log.Printf(">>>>>>>>> n: %d, a: %v, rtcpErr: %s", n, a, rtcpErr)
						continue
					}
					log.Printf(">>>>>>>>> n: %d, a: %v", n, a)
				}
			}()*/
		}
		//_, err = ReceiversWebrtcMap[sn].peerConnection.AddTrack(TracksMap[channel].Direction["Broadcast"].kind["audio"])
		//check(fName, sn, err)
		//_, err = ReceiversWebrtcMap[sn].peerConnection.AddTrack(TracksMap[channel].Direction["Broadcast"].kind["video"])
		//check(fName, sn, err)
	}
	ReceiversWebrtcMap[sn].actualChannel = channel
}
func registerReceiver(client *wss.Client) {
	var err error
	var fName string = "registerReceiver"
	var sn = client.Conn.RemoteAddr().String()
	log.Printf(" << REGISTER NEW RECEIVER << %s", sn)

	remoteP2PQueueMap[sn] = new(remoteP2PQueueConfig)
	remoteP2PQueueMap[sn].offer = make(chan offerConfig)
	remoteP2PQueueMap[sn].ice = make(chan iceConfig, 20)

	ReceiversWebrtcMap[sn] = new(SourceToWebrtcConfig)
	ReceiversWebrtcMap[sn].mediaEngine = &webrtc.MediaEngine{}
	ReceiversWebrtcMap[sn].settingEngine = &webrtc.SettingEngine{}
	ReceiversWebrtcMap[sn].interceptor = &interceptor.Registry{}
	ReceiversWebrtcMap[sn].receiverExitChan = make(chan bool)
	ReceiversWebrtcMap[sn].ssrcMap = make(map[string]string)
	ReceiversWebrtcMap[sn].actualChannel = ""

	if err := ReceiversWebrtcMap[sn].mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: codecMap["video"].MimeType, ClockRate: uint32(codecMap["video"].SampleRate) /*, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil*/},
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	if err := ReceiversWebrtcMap[sn].mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: codecMap["audio"].MimeType, ClockRate: uint32(codecMap["audio"].SampleRate) /*, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil*/},
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	if err := webrtc.RegisterDefaultInterceptors(ReceiversWebrtcMap[sn].mediaEngine, ReceiversWebrtcMap[sn].interceptor); err != nil {
		panic(err)
	}

	ReceiversWebrtcMap[sn].api = webrtc.NewAPI(
		webrtc.WithSettingEngine(*ReceiversWebrtcMap[sn].settingEngine),
		webrtc.WithMediaEngine(ReceiversWebrtcMap[sn].mediaEngine),
		webrtc.WithInterceptorRegistry(ReceiversWebrtcMap[sn].interceptor),
	)
	ReceiversWebrtcMap[sn].peerConnection, err = ReceiversWebrtcMap[sn].api.NewPeerConnection(webrtc.Configuration{ICEServers: []webrtc.ICEServer{}})
	check(fName, sn, err)

	jsonStr, _ := json.Marshal(ini.SCMap)
	data, _ := json.Marshal(&JsMessage{
		Request: "channelList",
		Data:    base64.URLEncoding.EncodeToString([]byte(jsonStr)),
	})
	client.Send <- data

	defer func() {
		log.Printf("unRegisterReceiver: %s", sn)
		ReceiversWebrtcMap[sn].peerConnection.Close()
		close(remoteP2PQueueMap[sn].ice)
		close(remoteP2PQueueMap[sn].offer)
		close(ReceiversWebrtcMap[sn].receiverExitChan)
		delete(ReceiversWebrtcMap, sn)
		delete(remoteP2PQueueMap, sn)
		client.Hub.Kill <- client
	}()
	for {
		select {
		case oc := <-remoteP2PQueueMap[sn].offer:
			var err error
			//log.Printf(" << OFFER << %s, channel: %s", sn, oc.offer)
			log.Printf(" << OFFER << %s, channel: %s", sn, oc.channel)

			ReceiversWebrtcMap[sn].peerConnection.Close()
			//ReceiversWebrtcMap[sn].peerConnection, err = ReceiversWebrtcMap[sn].api.NewPeerConnection(webrtcConfiguration)
			ReceiversWebrtcMap[sn].peerConnection, err = webrtc.NewPeerConnection(webrtcConfiguration)
			check(fName, sn, err)

			ReceiversWebrtcMap[sn].peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
				log.Printf("client - OnConnectionStateChange, sn: %s, State: %s\n", sn, state.String())
			})
			ReceiversWebrtcMap[sn].peerConnection.OnICECandidate(func(ice *webrtc.ICECandidate) {
				if ice == nil {
					return
				}
				ic := ice.ToJSON()
				log.Printf(" >> ICE >> Candidate: %s, SDPMLineIndex: %d", ic.Candidate, ic.SDPMLineIndex)
				candidateString, err := json.Marshal(ic)
				if err != nil {
					log.Println(err)
					return
				}
				data, _ := json.Marshal(&JsMessage{
					Request: "ice",
					Data:    base64.URLEncoding.EncodeToString([]byte(candidateString)),
				})
				client.Send <- data
			})
			ReceiversWebrtcMap[sn].peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
				log.Printf("client - OnICEConnectionStateChange, sn: %s, State: %s\n", sn, state.String())
			})
			ReceiversWebrtcMap[sn].peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
				log.Printf("client - OnICEGatheringStateChange, sn: %s, State: %s\n", sn, state.String())
			})
			ReceiversWebrtcMap[sn].peerConnection.OnNegotiationNeeded(func() {
				log.Printf("client - OnNegotiationNeeded, sn: %s", sn)
			})
			ReceiversWebrtcMap[sn].peerConnection.OnSignalingStateChange(func(state webrtc.SignalingState) {
				log.Printf("client - OnSignalingStateChange, sn: %s, State: %s\n", sn, state.String())
			})
			ReceiversWebrtcMap[sn].peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
				kind := track.Kind().String()
				log.Printf("registerReceiver - sn: %s, OnTrack track.Kind(): %s", sn, kind)
			})

			err = ReceiversWebrtcMap[sn].peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  oc.offer,
			})
			check(fName, sn, err)

			ReceiversWebrtcMap[sn].actualChannel = ""
			changeChannel(client, oc.channel)

			/*go func() {
				for {
					for _, sender := range ReceiversWebrtcMap[sn].peerConnection.GetSenders() {
						kind := sender.Track().Kind().String()
						pkts, a, err := sender.ReadRTCP()
						if err != nil {
							log.Printf(">> sender >> kind: %s, a: %v, err: %s", kind, a, err)
							break
						}
						for i, pkt := range pkts {
							log.Printf(">> sender >> kind: %s, pkt[%d], SSRC: %d, pkt: %v", kind, i, pkt.DestinationSSRC(), pkt)
						}
					}
				}
			}()*/

			answer, err := ReceiversWebrtcMap[sn].peerConnection.CreateAnswer(nil)
			check(fName, sn, err)

			var count int
			for _, line := range strings.Split(answer.SDP, "\n") {
				if line == "" {
					break
				}
				if strings.Index(line, "a=ssrc:") > -1 {
					if count == 0 {
						ReceiversWebrtcMap[sn].ssrcMap["video"] = line[7:strings.Index(line, " ")]
					} else if count == 4 {
						ReceiversWebrtcMap[sn].ssrcMap["audio"] = line[7:strings.Index(line, " ")]
						break
					}
					count++
				}
			}

			err = ReceiversWebrtcMap[sn].peerConnection.SetLocalDescription(answer)
			check(fName, sn, err)

			//log.Printf(" >> ANSWER >> %s", sdp)
			log.Printf(" >> ANSWER >>")

			data, _ := json.Marshal(&JsMessage{
				Request: "answer",
				Data:    base64.URLEncoding.EncodeToString([]byte(answer.SDP)),
			})
			client.Send <- data
		case ic := <-remoteP2PQueueMap[sn].ice:
			log.Printf(" << ICE << Candidate: %s, SDPMLineIndex: %d", ic.init.Candidate, ic.init.SDPMLineIndex)
			err := ReceiversWebrtcMap[sn].peerConnection.AddICECandidate(ic.init)
			check(fName, sn, err)
		case <-ReceiversWebrtcMap[sn].receiverExitChan:
			return
		default:
			<-time.After(sleepTime)
		}
	}
}
func unRegisterReceiver(client *wss.Client) {
	var sn = client.Conn.RemoteAddr().String()
	if _, ok := ReceiversWebrtcMap[sn]; !ok {
		return
	}
	ReceiversWebrtcMap[sn].receiverExitChan <- true
}

func main() {
	log.Println("Player START!")
	flag.Parse()

	codecMap = make(map[string]Codecs)
	updSourceMap = make(map[string]*updSource)
	ReceiversWebrtcMap = make(map[string]*SourceToWebrtcConfig)
	remoteP2PQueueMap = make(map[string]*remoteP2PQueueConfig)

	codecMap["video"] = Codecs{
		MimeType:      webrtc.MimeTypeH264,
		SampleRate:    90000,
		PacketMaxLate: 100,
		dep:           &codecs.H264Packet{},
	}
	codecMap["audio"] = Codecs{
		MimeType:      webrtc.MimeTypeOpus,
		SampleRate:    44100,
		PacketMaxLate: 10,
		dep:           &codecs.OpusPacket{},
	}

	go ini.ReadIniConfig()
	go core.StartCMDServer()
	go StartWebSocketServer()

	for {
		select {
		case sc := <-ini.StreamConfigChan:
			switch sc.Request {
			case core.AddRTPsourceRequest:
				AddRTPsource(sc)
			case core.DelRTPsourceRequest:
				DelRTPsource(sc)
			}
		case sc := <-core.StreamConfigChan:
			switch sc.Request {
			case core.AddRTPsourceRequest:
				ini.WriteSection(sc)
			case core.DelRTPsourceRequest:
				ini.DeleteSection(sc)
			}
		case <-time.After(sleepTime * 3):
		}
	}
}
