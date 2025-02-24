package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wangluozhe/chttp"
	"io"
	"net/url"
	"strings"
)

var settings = map[string]http.HTTP2SettingID{
	"HEADER_TABLE_SIZE":      http.HTTP2SettingHeaderTableSize,
	"ENABLE_PUSH":            http.HTTP2SettingEnablePush,
	"MAX_CONCURRENT_STREAMS": http.HTTP2SettingMaxConcurrentStreams,
	"INITIAL_WINDOW_SIZE":    http.HTTP2SettingInitialWindowSize,
	"MAX_FRAME_SIZE":         http.HTTP2SettingMaxFrameSize,
	"MAX_HEADER_LIST_SIZE":   http.HTTP2SettingMaxHeaderListSize,
	"UNKNOWN_SETTING_8":      http.HTTP2SettingID(8),
	"NO_RFC7540_PRIORITIES":  http.HTTP2SettingID(9),
}

type H2Settings struct {
	//HEADER_TABLE_SIZE
	//ENABLE_PUSH
	//MAX_CONCURRENT_STREAMS
	//INITIAL_WINDOW_SIZE
	//MAX_FRAME_SIZE
	//MAX_HEADER_LIST_SIZE
	Settings map[string]int `json:"Settings"`
	//HEADER_TABLE_SIZE
	//ENABLE_PUSH
	//MAX_CONCURRENT_STREAMS
	//INITIAL_WINDOW_SIZE
	//MAX_FRAME_SIZE
	//MAX_HEADER_LIST_SIZE
	SettingsOrder  []string                 `json:"SettingsOrder"`
	ConnectionFlow int                      `json:"ConnectionFlow"`
	HeaderPriority map[string]interface{}   `json:"HeaderPriority"`
	PriorityFrames []map[string]interface{} `json:"PriorityFrames"`
}

func ToHTTP2Settings(h2Settings *H2Settings) (http2Settings *http.HTTP2Settings) {
	http2Settings = &http.HTTP2Settings{
		Settings:       nil,
		ConnectionFlow: 0,
		HeaderPriority: &http.HTTP2PriorityParam{},
		PriorityFrames: nil,
	}
	if h2Settings.Settings != nil {
		if h2Settings.SettingsOrder != nil {
			for _, orderKey := range h2Settings.SettingsOrder {
				val := h2Settings.Settings[orderKey]
				if val != 0 || orderKey == "ENABLE_PUSH" {
					http2Settings.Settings = append(http2Settings.Settings, http.HTTP2Setting{
						ID:  settings[orderKey],
						Val: uint32(val),
					})
				}
			}
		} else {
			for id, val := range h2Settings.Settings {
				http2Settings.Settings = append(http2Settings.Settings, http.HTTP2Setting{
					ID:  settings[id],
					Val: uint32(val),
				})
			}
		}
	}
	if h2Settings.ConnectionFlow != 0 {
		http2Settings.ConnectionFlow = h2Settings.ConnectionFlow
	}
	if h2Settings.HeaderPriority != nil {
		var weight int
		var streamDep int
		w := h2Settings.HeaderPriority["weight"]
		switch w.(type) {
		case int:
			weight = w.(int)
		case float64:
			weight = int(w.(float64))
		}
		s := h2Settings.HeaderPriority["streamDep"]
		switch s.(type) {
		case int:
			streamDep = s.(int)
		case float64:
			streamDep = int(s.(float64))
		}
		var priorityParam *http.HTTP2PriorityParam
		if w == nil {
			priorityParam = &http.HTTP2PriorityParam{
				StreamDep: uint32(streamDep),
				Exclusive: h2Settings.HeaderPriority["exclusive"].(bool),
			}
		} else {
			priorityParam = &http.HTTP2PriorityParam{
				StreamDep: uint32(streamDep),
				Exclusive: h2Settings.HeaderPriority["exclusive"].(bool),
				Weight:    uint8(weight - 1),
			}
		}
		http2Settings.HeaderPriority = priorityParam
	}
	if h2Settings.PriorityFrames != nil {
		for _, frame := range h2Settings.PriorityFrames {
			var weight int
			var streamDep int
			var streamID int
			priorityParamSource := frame["priorityParam"].(map[string]interface{})
			w := priorityParamSource["weight"]
			switch w.(type) {
			case int:
				weight = w.(int)
			case float64:
				weight = int(w.(float64))
			}
			s := priorityParamSource["streamDep"]
			switch s.(type) {
			case int:
				streamDep = s.(int)
			case float64:
				streamDep = int(s.(float64))
			}
			sid := frame["streamID"]
			switch sid.(type) {
			case int:
				streamID = sid.(int)
			case float64:
				streamID = int(sid.(float64))
			}
			var priorityParam http.HTTP2PriorityParam
			if w == nil {
				priorityParam = http.HTTP2PriorityParam{
					StreamDep: uint32(streamDep),
					Exclusive: priorityParamSource["exclusive"].(bool),
				}
			} else {
				priorityParam = http.HTTP2PriorityParam{
					StreamDep: uint32(streamDep),
					Exclusive: priorityParamSource["exclusive"].(bool),
					Weight:    uint8(weight - 1),
				}
			}
			http2Settings.PriorityFrames = append(http2Settings.PriorityFrames, http.HTTP2PriorityFrame{
				HTTP2FrameHeader: http.HTTP2FrameHeader{
					StreamID: uint32(streamID),
				},
				HTTP2PriorityParam: priorityParam,
			})
		}
	}
	return http2Settings
}

func main() {
	get()
	post()
	post_json()
}

func request(req *http.Request) {
	h2s := &H2Settings{
		Settings: map[string]int{
			"HEADER_TABLE_SIZE":    65536,
			"ENABLE_PUSH":          0,
			"INITIAL_WINDOW_SIZE":  6291456,
			"MAX_HEADER_LIST_SIZE": 262144,
			//"MAX_CONCURRENT_STREAMS": 1000,
			//"MAX_FRAME_SIZE":         16384,
			"UNKNOWN_SETTING_8":     1,
			"NO_RFC7540_PRIORITIES": 1,
		},
		SettingsOrder: []string{
			"HEADER_TABLE_SIZE",
			"ENABLE_PUSH",
			"INITIAL_WINDOW_SIZE",
			"MAX_HEADER_LIST_SIZE",
			//"MAX_CONCURRENT_STREAMS",
			//"MAX_FRAME_SIZE",
			"UNKNOWN_SETTING_8",
			"NO_RFC7540_PRIORITIES",
		},
		ConnectionFlow: 15663105,
		HeaderPriority: map[string]interface{}{
			"weight":    256,
			"streamDep": 0,
			"exclusive": true,
		},
	}
	h2ss := ToHTTP2Settings(h2s)
	t1 := &http.Transport{}
	t2, err := http.HTTP2ConfigureTransports(t1)
	if err != nil {
		fmt.Println(err)
	}
	t2.HTTP2Settings = h2ss
	t1.H2Transport = t2
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	t1.Proxy = http.ProxyURL(proxyURL)
	client := http.Client{Transport: t1}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	text, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(text))
}

func get() {
	rawurl := "https://tls.peet.ws/api/all"
	req, _ := http.NewRequest("GET", rawurl, nil)
	headers := http.Header{
		"User-Agent":                []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36 Edg/117.0.2045.60"},
		"accept":                    []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"accept-language":           []string{"zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
		"accept-encoding":           []string{"gzip, deflate, br"},
		"upgrade-insecure-requests": []string{"1"},
		"sec-fetch-dest":            []string{"document"},
		"sec-fetch-mode":            []string{"navigate"},
		"sec-fetch-site":            []string{"none"},
		"sec-fetch-user":            []string{"?1"},
		"te":                        []string{"trailers"},
		http.PHeaderOrderKey: []string{
			":method",
			":authority",
			":scheme",
			":path",
		},
		http.HeaderOrderKey: []string{
			"user-agent",
			"accept",
			"accept-language",
			"accept-encoding",
			"upgrade-insecure-requests",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"te",
		},
	}
	req.Header = headers
	request(req)
}

func post() {
	rawurl := "https://tls.peet.ws/api/all"
	data := url.Values{}
	data.Set("username", "example_user")
	data.Set("password", "example_password")
	payload := strings.NewReader(data.Encode())
	req, _ := http.NewRequest("POST", rawurl, payload)
	headers := http.Header{
		"User-Agent":                []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36 Edg/117.0.2045.60"},
		"accept":                    []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"accept-language":           []string{"zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
		"accept-encoding":           []string{"gzip, deflate, br"},
		"upgrade-insecure-requests": []string{"1"},
		"sec-fetch-dest":            []string{"document"},
		"sec-fetch-mode":            []string{"navigate"},
		"sec-fetch-site":            []string{"none"},
		"sec-fetch-user":            []string{"?1"},
		"te":                        []string{"trailers"},
		http.PHeaderOrderKey: []string{
			":method",
			":authority",
			":scheme",
			":path",
		},
		http.HeaderOrderKey: []string{
			"user-agent",
			"accept",
			"accept-language",
			"accept-encoding",
			"upgrade-insecure-requests",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"te",
		},
	}
	req.Header = headers
	request(req)
}

func post_json() {
	rawurl := "https://tls.peet.ws/api/all"
	json_data := map[string]interface{}{
		"page":  "1",
		"limit": 10,
	}
	data, _ := json.Marshal(json_data)
	payload := bytes.NewReader(data)
	req, _ := http.NewRequest("POST", rawurl, payload)
	headers := http.Header{
		"User-Agent":                []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36 Edg/117.0.2045.60"},
		"accept":                    []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"accept-language":           []string{"zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2"},
		"accept-encoding":           []string{"gzip, deflate, br"},
		"upgrade-insecure-requests": []string{"1"},
		"sec-fetch-dest":            []string{"document"},
		"sec-fetch-mode":            []string{"navigate"},
		"sec-fetch-site":            []string{"none"},
		"sec-fetch-user":            []string{"?1"},
		"te":                        []string{"trailers"},
		http.PHeaderOrderKey: []string{
			":method",
			":authority",
			":scheme",
			":path",
		},
		http.HeaderOrderKey: []string{
			"user-agent",
			"accept",
			"accept-language",
			"accept-encoding",
			"upgrade-insecure-requests",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"te",
		},
	}
	req.Header = headers
	request(req)
}
