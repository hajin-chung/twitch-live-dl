const basicHeaders = {
	"Client-Id": "kimne78kx3ncx6brgo4mv6wki5h1ko",
	"User-Agent":
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
};

async function getPlaybackToken(login) {
	const payload = {
		operationName: "PlaybackAccessToken_Template",
		query:
			'query PlaybackAccessToken_Template($login: String!, $isLive: Boolean!, $vodID: ID!, $isVod: Boolean!, $playerType: String!) {  streamPlaybackAccessToken(channelName: $login, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isLive) {    value    signature   authorization { isForbidden forbiddenReasonCode }   __typename  }  videoPlaybackAccessToken(id: $vodID, params: {platform: "web", playerBackend: "mediaplayer", playerType: $playerType}) @include(if: $isVod) {    value    signature   __typename  }}',
		variables: {
			isLive: true,
			isVod: false,
			login,
			playerType: "site",
			vodID: "",
		},
	};
	const res = await fetch("https://gql.twitch.tv/gql", {
		method: "POST",
		headers: basicHeaders,
		body: JSON.stringify(payload),
	});
	const data = await res.json();
	const token = data.data.streamPlaybackAccessToken.value;
	const sig = data.data.streamPlaybackAccessToken.signature;
	if (typeof token !== "string" || typeof sig !== "string") {
		throw "bad playback token";
	} else {
		return { token, sig };
	}
}

async function getM3U8(login, token, sig) {
	const payload = {
		acmb: "e30=",
		allow_source: true,
		cdm: "wv",
		fast_bread: true,
		playlist_include_framerate: "true",
		reassignments_supported: "true",
		sig,
		supported_codecs: "avc1",
		token,
	};
	const params = new URLSearchParams(payload).toString();
	const url = new URL(
		`https://usher.ttvnw.net/api/channel/hls/${login}.m3u8?${params}`,
	);
	const res = await fetch(url.href);
	const body = await res.text();
	return body;
}

async function main() {
	const login = "972tv";
	const { token, sig } = await getPlaybackToken(login);
	const m3u8 = await getM3U8(login, token, sig);
	console.log(m3u8);
}

main().catch((e) => console.error(e));
