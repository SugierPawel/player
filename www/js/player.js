var Tools = function (m)
{
    "use strict";
    m.name = "Tools";
    m.switchers = new Object();
    m.date = new Object({
        "date": new Date(),
        "hours": "00",
        "minutes": "00",
        "seconds": "00"
    });
    m.serviceStatus = new Object({
        "map": false,
        "eventSource": window.EventSource ? true : false,
        "localStorage": window.localStorage ? true : false,
        "promise": window.Promise ? true : false,
        "webSocket": (window.WebSocket || window.MozWebSocket) ? true : false,
        "serviceWorker": false
    });
    m.GetTimeStamp = function () {
        return Tools.GetDate((new Date()).getTime());
    };
    m.GetDate = function (timestamp) {
        m.date["date"].setTime(timestamp);
        m.date["hours"] = m.date["date"].getHours();
        m.date["minutes"] = m.date["date"].getMinutes();
        m.date["seconds"] = m.date["date"].getSeconds();
        m.date["hours"] < 10 ? m.date["hours"] = "0" + m.date["hours"] : true;
        m.date["minutes"] < 10 ? m.date["minutes"] = "0" + m.date["minutes"] : true;
        m.date["seconds"] < 10 ? m.date["seconds"] = "0" + m.date["seconds"] : true;
        return m.date["hours"] + ":" + m.date["minutes"] + ":" + m.date["seconds"];
    };
    m.GetParams = function () {
        var params = new Object();
        var keyValues = location.search.slice(1).split("&");
        var decode = function (s) {
            return decodeURIComponent(s.replace(/\+/g, " "));
        };
        for (var i = 0; i < keyValues.length; ++i) {
            var key = keyValues[i].split("=");
            if (1 < key.length) params[decode(key[0])] = decode(key[1]);
        };
        return params;
    };
    m.Base64encode = function (str) {
        return btoa(encodeURIComponent(str).replace(/%([0-9A-F]{2})/g, function (match, p1) {
            return String.fromCharCode("0x" + p1);
        }));
    };
    m.GetMeta = function (el, name) {
        el = ("querySelector" in el) ? el : el.document;
        if (el.querySelector("meta[name=" + name + "]"))
            return el.querySelector("meta[name=" + name + "]").content;
        return undefined;
    };
    m.SetMeta = function (el, name, content) {
        el = ("querySelector" in el) ? el : el.document;
        if (!el.querySelector("meta[name=" + name + "]")) {
            let meta = el.createElement("meta");
            meta.setAttribute("name", name);
            el.getElementsByTagName("head")[0].appendChild(meta);
        }
        el.querySelector("meta[name=" + name + "]").content = content;
        return true;
    };
    m.GetClassValue = function (id, property, float) {
        if (!float) float = false;
        let value = window.getComputedStyle(document.getElementById(id)).getPropertyValue(property);
        if (value.length == 0) return undefined;
        return float ? parseFloat(value) : value;
    };
    m.GetLength = function (object) {
        if (!object) return 0;
        if (!m.serviceStatus["map"]) {
            var keys = 0;
            for (var i in object) if (object.hasOwnProperty(i)) keys++;
            return keys;
        } else {
            return Object.keys(object).length;
        };
    };
    m.GetFunctionArguments = function (module, method) {
        let str = window[module][method].toString();
        let replace = str.replace(/((\/\/.*$)|(\/\*[\s\S]*?\*\/)|(\s))/mg, "");
        let match = replace.match(/^function\s*[^\(]*\(\s*([^\)]*)\)/m)[1];
        let split = match.split(/,/);
        return split;
        /*return window[module][method].toString()
            .replace(/((\/\/.*$)|(\/\*[\s\S]*?\*\/)|(\s))/mg, "")
            .match(/^function\s*[^\(]*\(\s*([^\)]*)\)/m)[1]
            .split(/,/);*/
    };
    m.IsIeBrowser = function () {
        let ua = window.navigator.userAgent;
        return /MSIE|Trident|Edge/.test(ua);
    };
    m.IsFunc = function (func) {
        if (typeof func != "function") {
            Log.Write(Log.ERROR, "Podany parametr nie jest funkcją, typeof:" + typeof func);
            return false;
        };
        return true;
    };
    m.FromJson = function (str) {
        let obj = new Object();
        try {
            obj = JSON.parse(str)
        }
        catch (ex) {
            Log.Write(Log.ERROR, ex.message, str);
        };
        return obj;
    };
    m.ToJson = function (obj) {
        let json = "";
        try {
            json = JSON.stringify(obj)
        }
        catch (ex) {
            Log.Write(Log.ERROR, ex.message, obj);
        };
        return json;
    };
    m.CheckArguments = function (params)
    {
        let args = Array.prototype.slice.call(params);
        for (let i = 0; i < args.length; i++)
        {
            if (
                typeof args[i] != "boolean" &&
                typeof args[i] != "string" &&
                typeof args[i] != "number" &&
                !args[i])
            {
                Log.Write(Log.WARN, "parametr ma nieokreśloną wartość: ", args[i]);
                return false;
            };
        };
        return true;
    };
    m.ConvertToSpaces = function (str)
    {
        let spaces = "";
        for (let i = 0; i < str.length; i++)
            spaces += " ";
        return spaces;
    };
    m.GetRandomString = function (length) {
        let chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        let result = "";
        for (let i = 0; i < length; i++)
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        return result;
    };
    m.Init = function () {
        if (typeof MouseEvent !== "function") {
            (function () {
                window.MouseEvent = function (type, dict) {
                    dict = dict || {};
                    var event = document.createEvent("MouseEvents");
                    event.initMouseEvent(
                        type,
                        (typeof dict.bubbles == "undefined") ? true : !!dict.bubbles,
                        (typeof dict.cancelable == "undefined") ? false : !!dict.cancelable,
                        dict.view || window,
                        dict.detail | 0,
                        dict.screenX | 0,
                        dict.screenY | 0,
                        dict.clientX | 0,
                        dict.clientY | 0,
                        !!dict.ctrlKey,
                        !!dict.altKey,
                        !!dict.shiftKey,
                        !!dict.metaKey,
                        dict.button | 0,
                        dict.relatedTarget || undefined
                    );
                    return event;
                }
            })();
            MouseEvent.prototype = Event.prototype;
            window.MouseEvent = MouseEvent;
        };
        try {
            new window.Map().keys();
            m.serviceStatus["map"] = true;
        } catch (error) {
            m.serviceStatus["map"] = false;
        };
        if (typeof window.btoa != "function") {
            var digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";
            window.btoa = function (chars) {
                var buffer = "";
                var i, n;
                for (i = 0, n = chars.length; i < n; i += 3) {
                    var b1 = chars.charCodeAt(i) & 0xFF;
                    var b2 = chars.charCodeAt(i + 1) & 0xFF;
                    var b3 = chars.charCodeAt(i + 2) & 0xFF;
                    var d1 = b1 >> 2, d2 = ((b1 & 3) << 4) | (b2 >> 4);
                    var d3 = i + 1 < n ? ((b2 & 0xF) << 2) | (b3 >> 6) : 64;
                    var d4 = i + 2 < n ? (b3 & 0x3F) : 64;
                    buffer += digits.charAt(d1) + digits.charAt(d2) + digits.charAt(d3) + digits.charAt(d4);
                };
                return buffer;
            };
        };
        try {
            new Uint8Array(1);
        } catch (error) {
            var typed_array = function (arg) {
                var result;
                if (typeof arg === "number") {
                    result = new Array(arg);
                    for (var i = 0; i < arg; ++i) result[i] = 0;
                } else {
                    result = arg.slice(0);
                };
                result.buffer = result;
                result.byteLength = result.length;
                result.subarray = function (start, end) {
                    return this.slice(start, end);
                };
                result.set = function (array, offset) {
                    if (arguments.length < 2) offset = 0;
                    for (var i = 0, n = array.length; i < n; ++i, ++offset) this[offset] = array[i] & 0xFF;
                };
                if (typeof arg === "object" && arg.buffer) result.buffer = arg.buffer;
                return result;
            };
            window.Uint8Array = typed_array;
            window.Uint32Array = typed_array;
            window.Int32Array = typed_array;
        };
    };
    m.Init();
    return m;
};
var Log = function (m)
{
    "use strict";
    m.name = "Log";
    m.ERROR = "error";
    m.INFO = "info";
    m.WARN = "warn";
    m.textarea = undefined;
    m.CreateScreenLog = function () {
        m.textarea = document.createElement("textarea");
        m.textarea.setAttribute("id", "debugLogTextarea");
        m.textarea.style.display = "none";
        m.textarea.style.top = "50%";
        m.textarea.style.position = "fixed";
        m.textarea.style.width = "100%";//100vw
        m.textarea.style.height = "50%";//50vh
        m.textarea.style.color = "black";
        m.textarea.style.fontSize = "10px";
        m.textarea.style.fontWeight = "bold";
        m.textarea.style.padding = "5px 5px 5px 5px";
        m.textarea.style.textDecoration = "none";
        m.textarea.style.textAlign = "left";
        m.textarea.style.cursor = "pointer";
        m.textarea.style.zIndex = "9999";
        Engine.root.appendChild(m.textarea);
        return true;
    };
    m.Show = function () {
        m.textarea.style.display = "";
        return true;
    };
    m.Hide = function () {
        m.textarea.style.display = "none";
        return true;
    };
    m.debugLog = function (msg) {
        m.textarea.appendChild(document.createTextNode(msg + "\r\n"));
        m.textarea.scrollTop = m.textarea.scrollHeight;
        return true;
    };
    m.Write = function (type, msg, param) {
        let time = Tools.GetDate((new Date()).getTime());
        let module = "";
        let method = "";
        try {
            let stack = (new Error()).stack.split("\n")[2].trim().split(" ");
            let split = stack[1].split(".");
            module = split[1];
            method = split[3];
            if (split[1] == "eval") {
                method = stack[3].slice(0, -1);
                split = stack[stack.length - 1].split(":");
                if (split[1] == "10")
                    module = (new Error()).stack.split("\n")[3].trim().split(" ")[1].split(".")[1];
            };
        } catch (ex) {
        };
        let _msg = time + (module == "onerror" ? "" : " - " + module + "::" + method) + " - " + msg;
        let _param = !param ? "" : ", " + param;
        if (m.textarea !== undefined) m.debugLog(time + " - " + type + " - " + module + "::" + method + " - " + msg + _param);
        switch (type) {
            case m.WARN:
                param ? console.warn(_msg, param) : console.warn(_msg);
                break;
            case m.ERROR:
                param ? console.error(_msg, param) : console.error(_msg);
                break;
            case m.INFO:
                param ? console.log(_msg, param) : console.log(_msg);
                break;
            default:
                param ? console.log(_msg, param) : console.log(_msg);
                break;
        };
        window.onerror = function (message, url, line, column, error) {
            let _url = url.split("/");
            m.Write(LOG.ERROR, _url[3] + ", line: " + line + " >> " + error, message);
        };
        return true;
    };
    m.Init = function ()
    {
        m.CreateScreenLog();
        if (typeof window["Tools"].GetParams()["debug"] === "string") m.Show();
        return true;
    };
    m.Init();
    return m;
};
var Engine = function (m)
{
    "use strict";
    m.name = "Engine";
    m.modules = new Object();
    m.root = undefined;
    m.IsRegistered = function (module) {
        if (!window[module]) {
            LOG.Write(LOG.ERROR, "Nie ma takiego modułu:", module);
            return false;
        };
        return true;
    };
    m.RegisterModule = function (func) {
        if (typeof func != "function") {
            console.error("RegisterModule, typeof func:", func);
            return false;
        };
        let methods = func(new Object());
        for (let p in methods)
            if (typeof methods[p] == "function")
                methods[p].Name = p; //nadajemy nazwę dla poprawnego działania LOG.Help();
        let funcs = "";
        let name = m.name;
        window[name] = methods;
        m.modules[name] = true;
        if (funcs != "")
            return true;
        if (window["Log"] && typeof window["Log"].Write == "function")
        {
            Log.Write(Log.INFO, name);
        }
        else
        {
            console.log("RegisterModule:", name);
        };
        return true;
    };
    m.Init = function () {
        if (!document.getElementById("root"))
        {
            m.root = document.createElement("div");
            m.root.setAttribute("id", "root");
            m.root.setAttribute("class", "root");
            document.body.appendChild(m.root);
        }
        else {
            m.root = document.getElementById("root");
        };
        return true;
    };
    return m;
};
var Ws = function (m)
{
    "use strict";
    m.name = "Ws";
    m.bindedOnOpen = new Array();
    m.bindedOnClose = new Array();
    m.monitTimeoutId = undefined;
    m.monitActive = true;

    m.host = window.location.host;
    m.hostUrlPrefix = window.location.protocol == "chrome-extension:" ? "https:" : window.location.protocol === "https:" ? "https:" : "http:";
    m.protocol = window.location.protocol == "chrome-extension:" ? "wss" : window.location.protocol === "https:" ? "wss" : "ws";
    m.port = window.location.protocol == "chrome-extension:" ? "443" : window.location.protocol === "https:" ? "443" : "80";
    m.wsAddr = m.protocol + "://" + m.host + ":" + m.port + "/signal/";
    
    m.socket = null;
    m.CONNECTION_CLOSED = 0
    m.CONNECTION_PENDING = 1
    m.CONNECTION_OK = 2
    m.connectionStatus = m.CONNECTION_CLOSED;
    m.lastConnectTime = 0
    m.minConnectInterval = 10000
    m.connectTimeoutID = null;
    
    m._onopen = function (e)
    {
        Log.Write(Log.INFO, "socket _onopen, addr: " + e.target.url);
        m.connectionStatus = m.CONNECTION_OK;
        m.SetConnectionMonitState("connected");
        Player.p2psend();
        for (var i in m.bindedOnOpen)
            m.bindedOnOpen[i]();
    };
    m._onmessage = function (e)
    {
        try {
            var msg = JSON.parse(e.data);
            msg.data = atob(msg.data)
            Log.Write(Log.INFO, 'socket.onmessage', msg);
        } catch (ex) {
            Log.Write(Log.ERROR, 'socket onmessage::JSON.parse ex: ' + ex.message + ' - event.data: ' + e.data);
            return false;
        };
        switch (msg.request)
        {
            case 'addChannel':
                Player.AddChannel(msg.data);
                break;
            case 'delChannel':
                Player.DelChannel(msg.data);
                break;
            case 'channelList':
                Player.FillChannelsList(msg.data)
                break;
            case 'answer':
                WebRTC.SetRemoteDescription(msg.data);
                break;
            case 'ice':
                WebRTC.AddIceCandidate(msg.data);
                break;
        };
    };
    m._onclose = function (e) {
        Log.Write(Log.INFO, "socket _onclose, addr: " + e.target.url);
        m.connectionStatus = m.CONNECTION_CLOSED;
        m.SetConnectionMonitState("disconnected");
        m.Close();
        m.socket = null;
        m.Connect();
    };
    m._onerror = function (e) {
        Log.Write(Log.ERROR, "socket _onerror, addr: " + e.target.url + ", err: " + e.data);
    };

    m.Close = function ()
    {
        m.socket.close();
        for (var i in m.bindedOnClose)
            m.bindedOnClose[i]();
    };
    m.Open = function ()
    {
        switch (m.connectionStatus)
        {
            case m.CONNECTION_CLOSED:
                var _now = (new Date()).getTime();
                if (_now > m.lastConnectTime + Number(m.minConnectInterval))
                    break;
                var diff = m.wsLastConnecTime + Number(m.minConnectInterval) - _now;
                if (m.connectTimeoutID === null)
                    m.connectTimeoutID = setTimeout(m.connectTimeout, diff);
                return;
            case m.CONNECTION_PENDING:
            case m.CONNECTION_OK:
                m.SetConnectionMonitState("hide");
                return;
        };
        m.lastConnectTime = (new Date()).getTime();
        m.connectionStatus = m.CONNECTION_PENDING;
        m.SetConnectionMonitState("connecting");
        var WebSocket = window.WebSocket || window.MozWebSocket;
        if (!WebSocket)
        {
            Log.Write(Log.WARN, "socket - brak obsługi WebSocket");
            m.connectionStatus = m.CONNECTION_CLOSED;
            return;
        };
        try {
            Log.Write(Log.INFO, "socket łączę z: " + m.wsAddr);
            m.socket = new WebSocket(m.wsAddr);
        } catch (ex) {
            Log.Write(Log.ERROR, "socket błąd łączenia z: " + m.wsAddr + ", err: " + ex.message);
            m.connectionStatus = m.CONNECTION_CLOSED;
            return;
        };
        m.socket.onopen = m._onopen;
        m.socket.onclose = m._onclose;
        m.socket.onmessage = m._onmessage;
        m.socket.onerror = m._onerror;
        return true;
    };

    m.connectTimeout = function()
    {
        clearTimeout(m.connectTimeoutID);
        m.connectTimeoutID = null;
        m.Connect();
    };

    m.RegisterOnOpen = function (func) {
        if (!Tools.IsFunc(func))
            return false;
        m.bindedOnOpen.push(func);
        return true;
    };
    m.RegisterOnClose = function (func) {
        if (!Tools.IsFunc(func))
            return false;
        m.bindedOnClose.push(func);
        return true;
    };
    m.SetConnectionMonitState = function (state)
    {
        if (m.monitActive === false) return true;
        if (!document.getElementById("ConnectionMonit")) m.CreateConnectionMonit();
        var cm = document.getElementById("ConnectionMonit");
        var info_a = document.getElementById("info_a");
        cm.style.display = "block";
        switch (state)
        {
            case "disconnected":
                cm.style.backgroundColor = "#e71e1e";
                info_a.dataset.info = "!!!";
                break;
            case "connecting":
                cm.style.backgroundColor = "#c727d1";
                info_a.dataset.info = "...";
                break;
            case "connected":
                cm.style.backgroundColor = "#22b13d";
                info_a.dataset.info = "OK!";
                if (m.monitTimeoutId === undefined)
                {
                    m.monitTimeoutId = setTimeout(function ()
                    {
                        clearTimeout(m.monitTimeoutId);
                        m.monitTimeoutId = undefined;
                        cm.style.display = "none";
                    }, 1000);
                };
                break;
            case "hide":
                if (m.monitTimeoutId !== undefined)
                {
                    clearTimeout(m.monitTimeoutId);
                    m.monitTimeoutId = undefined;
                };
                cm.style.display = "none";
                break;
            default:
                break;
        };
        return true;
    };
    m.CreateConnectionMonit = function ()
    {
        if (m.monitActive === false) return true;
        var cm = document.createElement("div");
            cm.setAttribute("class", "ConnectionMonit");
            cm.setAttribute("id", "ConnectionMonit");
        document.getElementById("root").appendChild(cm);

        var a = document.createElement("a");
            a.setAttribute("class", "info_a");
            a.setAttribute("id", "info_a");
            a.dataset.info = "init";
        cm.appendChild(a);
        return true;
    };
    m.OnBeforeUnload = function (e)
    {
        var ret = "Zamykam połączenia z silnikiem ...";
        m.Close();
        e.returnValue = ret;
        return ret;
    };
    return m;
};
var WebRTC = function (m)
{
    "use strict";
    m.name = "WebRTC";
    m.requires = new Array();
    m.lastConnectTime = 0
    m.minConnectInterval = 15000
    m.connectTimeoutID = null;
    m.rtcpcConfiguration = {
        iceServers: [
            {
                url: "turn:172.26.9.100:5900?transport=udp",
                username: "turnserver",
                credential: "turnserver"
            },
            {
                url: "turn:172.26.9.100:5900?transport=tcp",
                username: "turnserver",
                credential: "turnserver"
            },
            {
                url: "turn:conf.polsat.com.pl:3478?transport=udp",
                username: "turnserver",
                credential: "turnserver"
            },
            {
                url: "turn:conf.polsat.com.pl:3478?transport=tcp",
                username: "turnserver",
                credential: "turnserver"
            },
            {
                url: "turn:conf.polsat.com.pl:433?transport=udp",
                username: "turnserver",
                credential: "turnserver"
            },
            {
                url: "turn:conf.polsat.com.pl:433?transport=tcp",
                username: "turnserver",
                credential: "turnserver"
            },
            {
                url: "stun:172.26.9.100:5900?transport=tcp"
            },
            {
                url: "stun:conf.polsat.com.pl:3478?transport=tcp"
            },
            {
                url: "stun:conf.polsat.com.pl:443?transport=tcp"
            }
        ]
    };
    m.rtcpc = null;
    m.stream = null;
    m.iceQueue = []
    
    m.SetRemoteDescription = function (sdp)
    {
        var _sdp = new RTCSessionDescription({type : 'answer', sdp : sdp});
        m.rtcpc.setRemoteDescription(_sdp).then(function()
        {
            Log.Write(Log.INFO, 'SetRemoteDescription', _sdp);
            while (m.iceQueue.length)
            {
                m.AddIceCandidate(m.iceQueue.shift())
            };
        }).catch(function(error)
        {
            Log.Write(Log.ERROR, 'SetRemoteDescription', error);
        });
    };
    
    m.AddIceCandidate = function(ice)
    {
        if (!m.rtcpc.remoteDescription)
        {
            m.iceQueue.push(ice)
            return
        };
        var jsonIce = JSON.parse(ice);
        var candidate = new RTCIceCandidate(jsonIce);
        m.rtcpc.addIceCandidate(candidate).then(function() {
            Log.Write(Log.INFO, 'AddIceCandidate', jsonIce);
            //m.rtcpc.addIceCandidate(candidate);
        }).catch(function(error)
        {
            Log.Write(Log.ERROR, 'AddIceCandidate ' + error, jsonIce);
        });
    };
    
    m.onAddStream = function(event)
    {
       var video_tracks = event.stream.getVideoTracks();
       var audio_tracks = event.stream.getAudioTracks();
       Log.Write(Log.INFO, 'onAddStream - video_tracks.length: ' + video_tracks.length + ' - audio_tracks.length: ' + audio_tracks.length, event);
    };
    
    m.onTrack = function(event)
    {
        Log.Write(Log.INFO, 'onTrack', event);
        m.stream = event.streams[0];
        Player.video.srcObject = event.streams[0];
    };
    
    m.onConnectionStateChange = function(event)
    {
        switch (event.target.connectionState)
        {
            case 'new':
            case 'checking':
            case 'connecting':
            case 'connected':
            case 'disconnected':
            case 'failed':
            case 'closed': 
                Log.Write(Log.INFO, 'onConnectionStateChange', m.info());
                break;
            default : 
                Log.Write(Log.WARN, 'onConnectionStateChange unknown state', m.info());
                break;
        };
        m.Connect();
    };
    
    m.onIceCandidate = function(event)
    {
        if (event.candidate == null) {
            Log.Write(Log.INFO, 'onIceCandidate null');
            return true;
        };
        var ice = JSON.stringify(event.candidate.toJSON());
        Log.Write(Log.INFO, 'onIceCandidate', event.candidate.toJSON());
        Player.SendToRemote('ice', ice);
    };
    
    m.onIceConnectionStateChange = function(event)
    {
        switch (event.target.iceConnectionState)
        {
            case 'new':
            case 'checking':
            case 'connected':
            case 'completed':
            case 'failed':
            case 'disconnected':
            case 'closed':
                Log.Write(Log.INFO, 'onIceConnectionStateChange', m.info());
                break;
            default:
                Log.Write(Log.WARN, 'onIceConnectionStateChange unknown state', m.info());
                break;
        };
    };
    
    m.onIceGatheringStateChange = function(event)
    {
        switch (event.target.iceGatheringState)
        {
            case 'gathering':
            case 'complete':
                Log.Write(Log.INFO, 'onIceGatheringStateChange', m.info());
                break;
            default:
                Log.Write(Log.WARN, 'onIceGatheringStateChange unknown state', m.info());
                break;
        };
    };
    
    m.onNegotiationNeeded = function(event)
    {
        Log.Write(Log.INFO, 'onNegotiationNeeded', m.info());
    };
    
    m.onSignalingStateChange = function(event)
    {
        switch (event.target.signalingState)
        {
            case 'closed':
                Log.Write(Log.INFO, 'onSignalingStateChange', m.info());
                break;
            case 'stable':
                Log.Write(Log.INFO, 'onSignalingStateChange', m.info());
                break;
            case 'have-local-offer':
                Log.Write(Log.INFO, 'onSignalingStateChange', m.info());
                break;
            case 'have-remote-offer':
                Log.Write(Log.INFO, 'onSignalingStateChange', m.info());
                break;
            case 'have-local-pranswer':
                Log.Write(Log.INFO, 'onSignalingStateChange', m.info());
                break;
            case 'have-remote-pranswer':
                Log.Write(Log.INFO, 'onSignalingStateChange', m.info());
                break;
            default:
                Log.Write(Log.WARN, 'onSignalingStateChange unknown state', m.info());
                break;
        };
    };
    
    m.connectTimeout = function()
    {
        clearTimeout(m.connectTimeoutID);
        m.connectTimeoutID = null;
        m.Connect();
    };
    
    m.Connect = function(force)
    {
        if (m.rtcpc && !force)
        {
            //if (m.rtcpc.connectionState == "failed" && m.rtcpc.iceGatheringState == "complete")
            //    return;
            switch (m.rtcpc.connectionState)
            {
                //case 'disconnected':
                //case 'closed':
                case 'failed':
                    var _now = (new Date()).getTime();
                    if (_now > m.lastConnectTime + Number(m.minConnectInterval))
                        break;
                    var diff = m.lastConnecTime + Number(m.minConnectInterval) - _now;
                    if (m.connectTimeoutID === null)
                        m.connectTimeoutID = setTimeout(m.connectTimeout, diff);
                    break;
                default:
                    return;
            };
        };
        m.Close();
        m.iceQueue = []
        
        m.rtcpc = new RTCPeerConnection(m.rtcpcConfiguration);
        m.rtcpc.onaddstream = m.onAddStream;
        m.rtcpc.ontrack = m.onTrack;
        m.rtcpc.onconnectionstatechange = m.onConnectionStateChange;
        m.rtcpc.onicecandidate = m.onIceCandidate;
        m.rtcpc.oniceconnectionstatechange = m.onIceConnectionStateChange;
        m.rtcpc.onicegatheringstatechange = m.onIceGatheringStateChange;
        m.rtcpc.onnegotiationneeded = m.onNegotiationNeeded;
        m.rtcpc.onsignalingstatechange = m.onSignalingStateChange;
        
        var init = {direction: "recvonly"}
        m.rtcpc.addTransceiver("video", init);
        m.rtcpc.addTransceiver("audio", init);
            
        Log.Write(Log.INFO, 'nawiązuję połączenie', m.info());
        
        m.rtcpc.createOffer({iceRestart:true})
        .then(function(offer)
        {
            return m.rtcpc.setLocalDescription(offer);
        })
        .then(function()
        { 
            Player.SendToRemote("offer", m.rtcpc.localDescription.sdp)
        })
        .catch(function(error)
        {
            Log.Write(Log.ERROR, "błąd wysyłania, err: " + error);
        });
    };
    
    m.Close = function()
    {
        if (m.rtcpc)
        {
            Log.Write(Log.INFO, 'Zamykam połączenie', m.info());
            const senders = m.rtcpc.getSenders();
            senders.forEach(function(sender)
            {
                if (sender.close)
                    sender.close();
                m.rtcpc.removeTrack(sender);
            });
            m.rtcpc.close();
            m.rtcpc = null;
        };
    };
    
    m.info = function()
    {
        var a = ' - connectionState: '    + (m.rtcpc ? m.rtcpc.connectionState    : 'unknown');
        var b = ' - iceConnectionState: ' + (m.rtcpc ? m.rtcpc.iceConnectionState : 'unknown');
        var c = ' - iceGatheringState: '  + (m.rtcpc ? m.rtcpc.iceGatheringState  : 'unknown');
        var d = ' - signalingState: '     + (m.rtcpc ? m.rtcpc.signalingState     : 'unknown');
        return a + b + c + d;
    };
    return m;
};
var Player = function (m)
{
    "use strict";
    m.name = "Player";
    
    m.local = new Object({offer:null,ice:new Array()});
    m.channel = null
    m.listMap = new Object();
    m.listSelect = null;
    m.video = null;

    m.p2psendICE = function()
    {
        switch(m.connectionStatus){
            case Ws.CONNECTION_CLOSED:
                Ws.Connect();
                break;
            case Ws.CONNECTION_PENDING:
                break;
            case Ws.CONNECTION_OK:
                var msg, ice, b;
                while (m.local.ice.length)
                {
                    ice = m.local.ice.shift();
                    try {
                        b = btoa(ice);
                        msg = JSON.stringify({request:"ice", data:b, channel:m.channel});
                    } catch (ex) {
                        Log.Write(Log.ERROR, "błąd kodowania ICE, ex: " + ex.message);
                        continue;
                    };
                    try{
                        Ws.socket.send(msg);
                    } catch (err){
                        Log.Write(Log.ERROR, "błąd wysyłania ICE, err: " + err);
                    };
                };
                break;
        };
    };
    m.p2psendOffer = function()
    {
        if (!m.local.offer)
            return;
        switch(m.connectionStatus){
            case m.CONNECTION_CLOSED:
                Ws.Connect();
                break;
            case Ws.CONNECTION_PENDING:
                break;
            case Ws.CONNECTION_OK:
                var msg, b;
                try {
                    b = btoa(m.local.offer);
                    msg = JSON.stringify({request:"offer", data:b, channel:m.channel});
                } catch (ex) {
                    Log.Write(Log.ERROR, "błąd kodowania oferty, ex: " + ex.message);
                    return;
                };
                try{
                    Ws.socket.send(msg);(msg);
                    m.local.offer = null;
                } catch (err){
                    Log.Write(Log.ERROR, "błąd wysyłania oferty, err: " + err);
                    return;
                };
                break;
        };
    };
    m.p2psend = function()
    {
        m.p2psendOffer();
        m.p2psendICE();
    };
    m.SendToRemote = function(request, data)
    {
        switch(request)
        {
            case "offer":
                Log.Write(Log.INFO, " >> SendToRemote >> request: " + request + ", channel: " + m.channel);
                m.local.offer = data
                m.local.ice = new Array();
                m.p2psendOffer();
                break;
            case "ice":
                Log.Write(Log.INFO, " >> SendToRemote >> request: " + request + ", data: " + data);
                m.local.ice.push(data);
                m.p2psendICE();
                break;
        };
    };
    
    m.listChange = function(e)
    {
        if (m.channel === e.target.value)
            return
        
        if (m.channel)
        {
            m.listMap[m.channel].option.removeAttribute("disabled");
            m.listMap[m.channel].option.removeAttribute("selected");
        }
        m.listMap[e.target.value].option.setAttribute("disabled", "");
        m.listMap[e.target.value].option.setAttribute("selected", "");
        
        m.channel = e.target.value;
        WebRTC.Connect(true);
        
        m.DelChannel(JSON.stringify({StreamName:"default"}));
    };
    m.AddChannel = function(data)
    {
        var channel = JSON.parse(data);
        if (m.listMap[channel.StreamName])
            return
        
        m.listMap[channel.StreamName] = new Object({
            StreamName:channel.StreamName,
            ChannelName:channel.ChannelName,
            option:document.createElement("option")
        });
        m.listMap[channel.StreamName].option.value = channel.StreamName;
        m.listMap[channel.StreamName].option.text = channel.ChannelName;
        m.listSelect.appendChild(m.listMap[channel.StreamName].option);
    };
    m.DelChannel = function(data)
    {
        var channel = JSON.parse(data);
        if (!m.listMap[channel.StreamName])
            return
        
        m.listSelect.removeChild(m.listMap[channel.StreamName].option);
        delete m.listMap[channel.StreamName]
    };
    m.FillChannelsList = function(data)
    {
        m.listMap["default"].option.text = "wybierz kanał!"
        m.listSelect.disabled = false;
        var list = JSON.parse(data);
        for (const ns in m.listMap)
            if (!list[ns] && ns !== "default")
                m.DelChannel(JSON.stringify(m.listMap[ns]));
        
        for (const ns in list)
        {
            if (!m.listMap[ns])
                m.AddChannel(JSON.stringify(list[ns]));
        };
    };

    m.Init = function()
    {
        var info = document.createElement('a');
        info.setAttribute('class', 'title_a');
        info.setAttribute('data-title', 'Player');
        Player.root.appendChild(info);

        var listPanel = document.createElement('div');
        listPanel.setAttribute('id', 'listPanel');
        listPanel.setAttribute('class', 'listPanel');
        Player.root.appendChild(listPanel);

        var playerPanel = document.createElement('div');
        playerPanel.setAttribute('id', 'playerPanel');
        playerPanel.setAttribute('class', 'playerPanel');
        Player.root.appendChild(playerPanel);

        m.listSelect = document.createElement("select");
        m.listSelect.setAttribute('id', 'channelsList');
        m.listSelect.setAttribute('class', 'channelsList');
        m.listSelect.disabled = true;
        m.listSelect.addEventListener('change', m.listChange);
        listPanel.appendChild(m.listSelect);
        
        m.video = document.createElement('video');
        m.video.autoplay = true;
        m.video.muted = false;
        m.video.controls = true;
        m.video.setAttribute('id', 'video');
        m.video.setAttribute('class', 'video');
        playerPanel.appendChild(m.video);

        m.AddChannel(JSON.stringify({StreamName:"default",ChannelName:"ładuję listę..."}));
        m.listMap["default"].option.setAttribute("selected", "");
    };
    m.Start = function()
    {
        m.Init();
        Ws.RegisterOnOpen(function()
        {
        });
        Ws.RegisterOnClose(function()
        {
            m.listSelect.disabled = true;
        });
        Ws.Open();
        return true;
    };
    return m;
};
!function ()
{
    window["Engine"] = Engine(new Object());
    Engine.Init();
    Engine.RegisterModule(WebRTC);
    Engine.RegisterModule(Tools);
    Engine.RegisterModule(Log);
    Engine.RegisterModule(Ws);
    Engine.RegisterModule(Player);
    Player.Start();
}();