package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type CacheItem struct {
	url string
	ttl time.Time
}

type Cache map[string]*CacheItem

func main() {
	urlCache := Cache{}
	app := fiber.New()
	app.Use(logger.New())

	app.Get("/cache", func(c *fiber.Ctx) error {
		data, err := json.Marshal(urlCache)
		if err != nil {
			return c.Status(500).SendString("cannot marshal cache")
		}

		return c.Send(data)
	})

	app.Get("/:login.m3u8", func(c *fiber.Ctx) error {
		login := c.Params("login")
		var url string

		cache := urlCache[login]
		if cache != nil && time.Now().Compare(cache.ttl) == -1 {
			url = cache.url
		} else {
			playbackToken, err := GetPlaybackToken(login)
			if err != nil {
				return c.Status(500).SendString("bad login")
			}

			url = BuildHlsUrl(login, *playbackToken)
			urlCache[login] = &CacheItem{
				url: url,
				ttl: time.Now().Add(time.Hour * 16),
			}
		}

		res, err := http.Get(url)
		if err != nil {
			return c.Status(500).SendString("bad m3u8 url")
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return c.Status(500).SendString("cannot read body")
		}

		return c.Type("application/vnd.apple.mpegurl").Send(body)
	})

	app.Listen(":8838")
}

type PlaybackToken struct {
	Token string
	Sig   string
}

type PlaybackTokenVariables struct {
	IsLive     bool   `json:"isLive"`
	IsVod      bool   `json:"isVod"`
	Login      string `json:"login"`
	PlayerType string `json:"playerType"`
	VodId      string `json:"vodID"`
}

type PlaybackTokenPayload struct {
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     PlaybackTokenVariables `json:"variables"`
}

func GetPlaybackToken(login string) (*PlaybackToken, error) {
	payload := &PlaybackTokenPayload{
		OperationName: "PlaybackAccessToken_Template",
		Query:         `query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}`,
		Variables: PlaybackTokenVariables{
			IsLive:     true,
			IsVod:      false,
			Login:      login,
			PlayerType: "site",
			VodId:      "",
		},
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(
		"POST", "https://gql.twitch.tv/gql", bytes.NewBuffer(payloadJson))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Client-Id", "kimne78kx3ncx6brgo4mv6wki5h1ko")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	data, ok := data["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("type conversion")
	}

	stream, ok := data["streamPlaybackAccessToken"].(map[string]interface{})
	if !ok {
		return nil, errors.New("type conversion")
	}

	token := stream["value"].(string)
	sig := stream["signature"].(string)
	return &PlaybackToken{token, sig}, nil
}

func BuildHlsUrl(login string, token PlaybackToken) string {
	params := url.Values{}
	params.Add("acmb", "e30=")
	params.Add("allow_source", "true")
	params.Add("cdm", "wv")
	params.Add("fast_bread", "true")
	params.Add("playlist_include_framerate", "true")
	params.Add("reassignments_supported", "true")
	params.Add("sig", token.Sig)
	params.Add("supported_codecs", "avc1")
	params.Add("token", token.Token)
	url := fmt.Sprintf(
		"https://usher.ttvnw.net/api/channel/hls/%s.m3u8?%s",
		login, params.Encode())
	return url
}
