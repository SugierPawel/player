//go:build !js
// +build !js

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
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
	sleepTime     = time.Millisecond * 100
	webSocketAddr = "172.26.9.100:2000" //"172.26.9.100:2000" "127.0.0.1:2000"
)

var wssHub *wss.Hub
var sourceMutex sync.Mutex
var updSourceMap map[string]*updSource
var TracksMap map[string]*TracksConfig
var SourceToWebrtcMap map[string]*SourceToWebrtcConfig
var ReceiversWebrtcMap map[string]*SourceToWebrtcConfig
var remoteP2PQueueMap map[string]*remoteP2PQueueConfig
var codecMap map[string]Codecs

type updSource struct {
	rtcpConn *net.UDPConn
	rtpConn  *net.UDPConn
	wg       *sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
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
	ssrc                 webrtc.SSRC
	receiverExitChan     chan bool
}
type JsMessage struct {
	Request string `json:"request"`
	Data    string `json:"data"`
	Channel string `json:"channel"`
}
type TracksDirectionConfig struct {
	depacketizer map[string]rtp.Depacketizer
	sampleBuffer map[string]*samplebuilder.SampleBuilder
	kind         map[string]*webrtc.TrackLocalStaticSample
}
type TracksConfig struct {
	Direction map[string]*TracksDirectionConfig
}
type Codecs struct {
	MimeType      string
	SampleRate    int
	PacketMaxLate int
	dep           rtp.Depacketizer
}

var receiverWebrtcConfiguration = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs:       []string{"turn:172.26.9.100:5900"},
			Username:   "turnserver",
			Credential: "turnserver",
		},
		{
			URLs: []string{"stun:172.26.9.100:5900"},
		},
	},
}
var webrtcConfiguration = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:172.26.9.100:5900"},
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

	TracksMap[sn] = new(TracksConfig)
	TracksMap[sn].Direction = make(map[string]*TracksDirectionConfig)
	updSourceMap[sn] = new(updSource)
	SourceToWebrtcMap[sn] = new(SourceToWebrtcConfig)

	initLocalTracks(sc, "RTP")
	initLocalTracks(sc, "Broadcast")

	updSourceMap[sn].wg = &sync.WaitGroup{}
	updSourceMap[sn].ctx, updSourceMap[sn].cancel = context.WithCancel(context.Background())

	go updSourceMap[sn].InitRtcp(sc)
	go updSourceMap[sn].InitRtp(sc)

	sdp1 := make(chan string, 1)
	sdp2 := make(chan string, 1)
	ice1 := make(chan *webrtc.ICECandidate, 1)
	ice2 := make(chan *webrtc.ICECandidate, 1)

	go updSourceMap[sn].localRTPofferer(sn, sdp1, sdp2, ice1, ice2)
	go updSourceMap[sn].localRTPanswerer(sn, sdp1, sdp2, ice1, ice2)

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

	updSourceMap[sn].wg.Add(4)
	updSourceMap[sn].cancel()
	updSourceMap[sn].wg.Wait()

	delete(TracksMap, sn)
	delete(updSourceMap, sn)
}

func initLocalTracks(sc *core.StreamConfig, direction string) {
	sn := sc.StreamName
	var err error

	TracksMap[sn].Direction[direction] = new(TracksDirectionConfig)
	TracksMap[sn].Direction[direction].kind = make(map[string]*webrtc.TrackLocalStaticSample)
	TracksMap[sn].Direction[direction].depacketizer = make(map[string]rtp.Depacketizer)
	TracksMap[sn].Direction[direction].sampleBuffer = make(map[string]*samplebuilder.SampleBuilder)

	for kind := range codecMap {
		TracksMap[sn].Direction[direction].depacketizer[kind] = codecMap[kind].dep
		TracksMap[sn].Direction[direction].sampleBuffer[kind] = samplebuilder.New(
			uint16(codecMap[kind].PacketMaxLate),
			TracksMap[sn].Direction[direction].depacketizer[kind],
			uint32(codecMap[kind].SampleRate))
		TracksMap[sn].Direction[direction].kind[kind], err = webrtc.NewTrackLocalStaticSample(
			webrtc.RTPCodecCapability{MimeType: codecMap[kind].MimeType},
			kind,
			sc.ChannelName)
		if err != nil {
			log.Printf("initLocalTracks, sn: %s, kind: %s, direction: %s, error: %s", sn, kind, direction, err)
		}
	}
}

func (l *updSource) InitRtcp(sc *core.StreamConfig) {
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

	updSourceMap[sn].rtcpConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(IPIn), Port: rtcpPort})
	if err != nil {
		log.Printf("InitRtcp, sn: %s, IPIn: %s, port: %d, err: %s", sn, IPIn, rtcpPort, err)
		return
	}
	for {
		select {
		case <-l.ctx.Done():
			log.Printf("InitRtcp, sn: %s, ctx.Done()", sn)
			updSourceMap[sn].rtcpConn.Close()
			return
		default:
			p := make([]byte, 1200)
			rtcpN, _, err := updSourceMap[sn].rtcpConn.ReadFrom(p)
			if err != nil {
				log.Printf("InitRtcp, sn: %s, ReadFrom error: %s", sn, err)
				break
			}
			packets, err := rtcp.Unmarshal(p[:rtcpN])
			for n, packet := range packets {
				log.Printf("InitRtcp << sn: %s, n: %d, SSRC: %d", sn, n, packet.DestinationSSRC())
			}
			for rec, config := range ReceiversWebrtcMap {
				if config.actualChannel == sn {

					for _, sender := range config.peerConnection.GetSenders() {
						switch sender.Track().Kind().String() {
						case "video":

						case "audio":

						}

						s, err := sender.Transport().WriteRTCP(packets)
						if err != nil {
							log.Printf("InitRtcp << błąd wysłania do rec: %s, err: %s", rec, err)
						} else {
							log.Printf("InitRtcp << wysłano: %d, do rec: %s, sender.Track().StreamID(): %s", s, rec, sender.Track().StreamID())
						}

					}

					//err = config.peerConnection.WriteRTCP(packets)

				}
			}

		}
	}
}
func (l *updSource) InitRtp(sc *core.StreamConfig) {
	defer func() {
		l.wg.Done()
	}()
	sn := sc.StreamName
	var err error
	IPIn := ini.SCMap[sn].IPIn
	var port = ini.SCMap[sn].PortIn
	var broadcast string = "Broadcast"

	updSourceMap[sn].rtpConn, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(IPIn), Port: port})
	if err != nil {
		log.Printf("InitRtp, sn: %s, IPIn: %s, port: %d, err: %s", sn, IPIn, port, err)
		return
	}
	var kind string
	for {
		select {
		case <-l.ctx.Done():
			log.Printf("rtpConn, sn: %s, ctx.Done()", sn)
			updSourceMap[sn].rtpConn.Close()
			return
		default:
			packet := make([]byte, 1200)
			rtpPacket := &rtp.Packet{}
			n, _, err := updSourceMap[sn].rtpConn.ReadFrom(packet)
			if err != nil {
				log.Printf("InitRtp, sn: %s, ReadFrom error: %s", sn, err)
				break
			}
			if err = rtpPacket.Unmarshal(packet[:n]); err != nil {
				log.Printf("InitRtp, sn: %s, rtpPacket.Unmarshal error: %s", sn, err)
				break
			}

			switch rtpPacket.Header.PayloadType {
			case 96:
				kind = "video"
			case 97:
				kind = "audio"
			}
			log.Printf("InitRtp <<<< kind: %s, n: %d, pt: %d, SSRC: %d", kind, n, rtpPacket.Header.PayloadType, rtpPacket.SSRC)
			TracksMap[sn].Direction[broadcast].sampleBuffer[kind].Push(rtpPacket)
			for {
				sample := TracksMap[sn].Direction[broadcast].sampleBuffer[kind].Pop()
				if sample == nil {
					//log.Printf("InitRtp << nie gotowy...., kind: %s", kind)
					break
				}
				//log.Printf("InitRtp >> WriteSample!!!, kind: %s, ts: %d, dropped: %d", kind, sample.PacketTimestamp, sample.PrevDroppedPackets)
				if err := TracksMap[sn].Direction[broadcast].kind[kind].WriteSample(*sample); err != nil {
					log.Printf("InitRtp, kind: %s, sn: %s, WriteSample error: %s", kind, sn, err)
				}
			}
		}
	}
}

func (l *updSource) localRTPofferer(sn string, offerSDP chan<- string, answerSDP <-chan string, iceOffer chan<- *webrtc.ICECandidate, iceAnswer <-chan *webrtc.ICECandidate) {
	defer func() {
		l.wg.Done()
	}()
	var fName string = "localRTPofferer"
	var err error

	SourceToWebrtcMap[sn].mediaEngine = &webrtc.MediaEngine{}
	SourceToWebrtcMap[sn].settingEngine = &webrtc.SettingEngine{}
	SourceToWebrtcMap[sn].interceptor = &interceptor.Registry{}

	if err := SourceToWebrtcMap[sn].mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: codecMap["video"].MimeType, ClockRate: uint32(codecMap["video"].SampleRate), Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	if err := SourceToWebrtcMap[sn].mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: codecMap["audio"].MimeType, ClockRate: uint32(codecMap["audio"].SampleRate), Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        97,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	if err := webrtc.RegisterDefaultInterceptors(SourceToWebrtcMap[sn].mediaEngine, SourceToWebrtcMap[sn].interceptor); err != nil {
		panic(err)
	}
	SourceToWebrtcMap[sn].api = webrtc.NewAPI(
		webrtc.WithSettingEngine(*SourceToWebrtcMap[sn].settingEngine),
		webrtc.WithMediaEngine(SourceToWebrtcMap[sn].mediaEngine),
		webrtc.WithInterceptorRegistry(SourceToWebrtcMap[sn].interceptor),
	)

	SourceToWebrtcMap[sn].peerConnection, err = SourceToWebrtcMap[sn].api.NewPeerConnection(webrtcConfiguration)
	if err != nil {
		panic(err)
	}

	SourceToWebrtcMap[sn].peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("OnConnectionStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].peerConnection.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			return
		}
		candidateString, err := json.Marshal(ice.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}
		log.Default().Printf(" >> rtp ICE >> %s", candidateString)
		iceOffer <- ice
	})
	SourceToWebrtcMap[sn].peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("localRTPofferer - OnICEConnectionStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("localRTPofferer - OnICEGatheringStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].peerConnection.OnNegotiationNeeded(func() {
		log.Printf("localRTPofferer - OnNegotiationNeeded, sn: %s", sn)
	})
	SourceToWebrtcMap[sn].peerConnection.OnSignalingStateChange(func(state webrtc.SignalingState) {
		log.Printf("localRTPofferer - OnSignalingStateChange, sn: %s, state: %s\n", sn, state.String())
	})

	_, err = SourceToWebrtcMap[sn].peerConnection.AddTransceiverFromTrack(TracksMap[sn].Direction["RTP"].kind["video"],
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		},
	)
	if err != nil {
		panic(err)
	}
	_, err = SourceToWebrtcMap[sn].peerConnection.AddTransceiverFromTrack(TracksMap[sn].Direction["RTP"].kind["audio"],
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		},
	)
	if err != nil {
		panic(err)
	}

	offer, err := SourceToWebrtcMap[sn].peerConnection.CreateOffer(nil)
	check(fName, sn, err)

	//log.Printf(" >> rtp OFFER >>\n%v\n", offer.SDP)
	log.Printf(" >> rtp OFFER >>")

	err = SourceToWebrtcMap[sn].peerConnection.SetLocalDescription(offer)
	check(fName, sn, err)

	offerSDP <- offer.SDP

	answer := <-answerSDP
	//log.Printf(" << rtp ANSWER <<\n%v\n", answer)
	log.Printf(" << rtp ANSWER <<")

	err = SourceToWebrtcMap[sn].peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer,
	})
	check(fName, sn, err)

	for {
		select {
		case <-l.ctx.Done():
			log.Printf("localRTPofferer, sn: %s, ctx.Done()", sn)
			return
		case ice := <-iceAnswer:
			err = SourceToWebrtcMap[sn].peerConnection.AddICECandidate(ice.ToJSON())
			check(fName, sn, err)
		default:
			<-time.After(sleepTime)
		}
	}
}
func (l *updSource) localRTPanswerer(sn string, offerSDP <-chan string, answerSDP chan<- string, iceOffer <-chan *webrtc.ICECandidate, iceAnswer chan<- *webrtc.ICECandidate) {
	defer func() {
		l.wg.Done()
	}()
	var fName string = "localRTPanswerer"
	var err error
	SourceToWebrtcMap[sn].answerPeerConnection, err = webrtc.NewPeerConnection(webrtcConfiguration)
	check(fName, sn, err)

	SourceToWebrtcMap[sn].answerPeerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("localRTPanswerer - OnConnectionStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].answerPeerConnection.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			return
		}
		init := ice.ToJSON()
		log.Printf(" << rtp ICE << Candidate: %s, SDPMLineIndex: %d", init.Candidate, init.SDPMLineIndex)
		iceAnswer <- ice
	})
	SourceToWebrtcMap[sn].answerPeerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("localRTPanswerer - OnICEConnectionStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].answerPeerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("localRTPanswerer - OnICEGatheringStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].answerPeerConnection.OnNegotiationNeeded(func() {
		log.Printf("localRTPanswerer - OnNegotiationNeeded, sn: %s", sn)
	})
	SourceToWebrtcMap[sn].answerPeerConnection.OnSignalingStateChange(func(state webrtc.SignalingState) {
		log.Printf("localRTPanswerer - OnSignalingStateChange, sn: %s, state: %s\n", sn, state.String())
	})
	SourceToWebrtcMap[sn].answerPeerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		kind := track.Kind().String()
		log.Printf("answerer - sn: %s, OnTrack track.Kind(): %s", sn, kind)
	})

	err = SourceToWebrtcMap[sn].answerPeerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  <-offerSDP,
	})
	check(fName, sn, err)

	answer, err := SourceToWebrtcMap[sn].answerPeerConnection.CreateAnswer(nil)
	check(fName, sn, err)

	err = SourceToWebrtcMap[sn].answerPeerConnection.SetLocalDescription(answer)
	check(fName, sn, err)

	answerSDP <- answer.SDP

	for {
		select {
		case <-l.ctx.Done():
			log.Printf("localRTPanswerer, sn: %s, ctx.Done()", sn)
			return
		case ice := <-iceOffer:
			err = SourceToWebrtcMap[sn].answerPeerConnection.AddICECandidate(ice.ToJSON())
			check(fName, sn, err)
		default:
			<-time.After(sleepTime)
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
	var err error
	if _, ok := SourceToWebrtcMap[channel]; !ok {
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
		for _, sender := range SourceToWebrtcMap[sn].peerConnection.GetSenders() {

			kind := sender.Track().Kind().String()

			//err = SourceToWebrtcMap[sn].peerConnection.RemoveTrack(sender)
			//check(fName, sn, err)
			//_, err = SourceToWebrtcMap[sn].peerConnection.AddTrack(TracksMap[channel].Direction["Broadcast"].kind[kind])
			//check(fName, sn, err)

			err = sender.ReplaceTrack(TracksMap[channel].Direction["Broadcast"].kind[kind])
			check(fName, sn, err)

			/*
				Do prawidłowego działania "ReplaceTrack" należy zadbać o synchronizacje SequenceNumber/Timestamp...
				err = sender.ReplaceTrack(TracksMap[channel].Direction["Broadcast"].kind[sender.Track().Kind().String()])
				check(fName, sn, err)
				switch sender.Track().Kind().String() {
				case "video":
					err = SourceToWebrtcMap[sn].peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(SourceToWebrtcMap[sn].ssrc)}})
					check(fName, sn, err)
				case "audio":
				}*/
		}
	} else if ReceiversWebrtcMap[sn].actualChannel == "" {
		log.Printf(" << ASIGN CHANNEL << klient: %s, kanał: %s", sn, channel)
		for _, track := range TracksMap[channel].Direction["Broadcast"].kind {
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
			log.Printf(" << OFFER << %s, channel: %s", sn, oc.channel)

			ReceiversWebrtcMap[sn].peerConnection.Close()
			ReceiversWebrtcMap[sn].peerConnection, err = ReceiversWebrtcMap[sn].api.NewPeerConnection(receiverWebrtcConfiguration)
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

			answer, err := ReceiversWebrtcMap[sn].peerConnection.CreateAnswer(nil)
			check(fName, sn, err)
			err = ReceiversWebrtcMap[sn].peerConnection.SetLocalDescription(answer)
			check(fName, sn, err)

			log.Printf(" >> ANSWER >> %s", answer.Type.String())

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
	SourceToWebrtcMap = make(map[string]*SourceToWebrtcConfig)
	ReceiversWebrtcMap = make(map[string]*SourceToWebrtcConfig)
	remoteP2PQueueMap = make(map[string]*remoteP2PQueueConfig)
	TracksMap = make(map[string]*TracksConfig)

	codecMap["video"] = Codecs{
		MimeType:      webrtc.MimeTypeH264,
		SampleRate:    90000,
		PacketMaxLate: 500,
		dep:           &codecs.H264Packet{},
	}
	codecMap["audio"] = Codecs{
		MimeType:      webrtc.MimeTypeOpus,
		SampleRate:    48000,
		PacketMaxLate: 500,
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
