package appid

import (
	"strings"
	"sync"
)

// AppCategory represents a category of applications.
type AppCategory string

const (
	CategoryStreamingMedia   AppCategory = "streaming_media"
	CategorySocialNetworking AppCategory = "social_networking"
	CategoryMessaging        AppCategory = "messaging"
	CategoryGames            AppCategory = "games"
	CategoryVPNTunneling     AppCategory = "vpn_tunneling"
	CategoryFileTransfer     AppCategory = "file_transfer"
	CategoryBusiness         AppCategory = "business"
	CategoryAds              AppCategory = "ads"
	CategoryUnknown          AppCategory = "unknown"
)

// AppDefinition defines an application and its detection patterns.
type AppDefinition struct {
	Name        string      // Canonical app name (e.g., "netflix")
	DisplayName string      // Human-readable name (e.g., "Netflix")
	Category    AppCategory // Application category
	Hostnames   []string    // Known hostnames (wildcards supported)
}

// Detector provides application detection functionality.
// This is a fallback implementation using hostname matching.
// For full DPI, use the nDPI CGO wrapper (ndpi.go).
type Detector struct {
	mu   sync.RWMutex
	apps map[string]*AppDefinition // name -> definition
	// Hostname index for fast lookup
	hostnameIndex map[string]string // hostname suffix -> app name
}

// NewDetector creates a new application detector with built-in definitions.
func NewDetector() *Detector {
	d := &Detector{
		apps:          make(map[string]*AppDefinition),
		hostnameIndex: make(map[string]string),
	}

	// Load built-in app definitions
	d.loadBuiltinApps()

	return d
}

// loadBuiltinApps loads the built-in application definitions.
func (d *Detector) loadBuiltinApps() {
	apps := []AppDefinition{
		// Streaming Media
		{Name: "netflix", DisplayName: "Netflix", Category: CategoryStreamingMedia,
			Hostnames: []string{"netflix.com", "nflxvideo.net", "nflximg.net", "nflxso.net", "nflxext.com"}},
		{Name: "youtube", DisplayName: "YouTube", Category: CategoryStreamingMedia,
			Hostnames: []string{"youtube.com", "googlevideo.com", "ytimg.com", "youtu.be", "youtube-nocookie.com", "yt3.ggpht.com"}},
		{Name: "spotify", DisplayName: "Spotify", Category: CategoryStreamingMedia,
			Hostnames: []string{"spotify.com", "scdn.co", "spotifycdn.com", "spoti.fi"}},
		{Name: "twitch", DisplayName: "Twitch", Category: CategoryStreamingMedia,
			Hostnames: []string{"twitch.tv", "twitchcdn.net", "ttvnw.net", "jtvnw.net"}},
		{Name: "disney_plus", DisplayName: "Disney+", Category: CategoryStreamingMedia,
			Hostnames: []string{"disneyplus.com", "disney-plus.net", "dssott.com", "bamgrid.com"}},
		{Name: "amazon_video", DisplayName: "Amazon Prime Video", Category: CategoryStreamingMedia,
			Hostnames: []string{"primevideo.com", "amazonvideo.com", "aiv-cdn.net", "aiv-delivery.net"}},
		{Name: "hbo_max", DisplayName: "HBO Max", Category: CategoryStreamingMedia,
			Hostnames: []string{"hbomax.com", "max.com", "hbo.com"}},
		{Name: "apple_tv", DisplayName: "Apple TV+", Category: CategoryStreamingMedia,
			Hostnames: []string{"tv.apple.com", "apple.com/tv"}},
		{Name: "deezer", DisplayName: "Deezer", Category: CategoryStreamingMedia,
			Hostnames: []string{"deezer.com", "dzcdn.net"}},
		{Name: "soundcloud", DisplayName: "SoundCloud", Category: CategoryStreamingMedia,
			Hostnames: []string{"soundcloud.com", "sndcdn.com"}},
		{Name: "tidal", DisplayName: "Tidal", Category: CategoryStreamingMedia,
			Hostnames: []string{"tidal.com", "tidalhifi.com"}},
		{Name: "vimeo", DisplayName: "Vimeo", Category: CategoryStreamingMedia,
			Hostnames: []string{"vimeo.com", "vimeocdn.com"}},
		{Name: "dailymotion", DisplayName: "Dailymotion", Category: CategoryStreamingMedia,
			Hostnames: []string{"dailymotion.com", "dmcdn.net"}},

		// Social Networking
		{Name: "facebook", DisplayName: "Facebook", Category: CategorySocialNetworking,
			Hostnames: []string{"facebook.com", "fbcdn.net", "fb.com", "fb.me", "facebook.net", "fbsbx.com"}},
		{Name: "instagram", DisplayName: "Instagram", Category: CategorySocialNetworking,
			Hostnames: []string{"instagram.com", "cdninstagram.com", "instagr.am"}},
		{Name: "twitter", DisplayName: "Twitter/X", Category: CategorySocialNetworking,
			Hostnames: []string{"twitter.com", "x.com", "twimg.com", "t.co", "tweetdeck.com"}},
		{Name: "tiktok", DisplayName: "TikTok", Category: CategorySocialNetworking,
			Hostnames: []string{"tiktok.com", "tiktokcdn.com", "tiktokv.com", "musical.ly", "byteoversea.com", "ibytedtos.com"}},
		{Name: "linkedin", DisplayName: "LinkedIn", Category: CategorySocialNetworking,
			Hostnames: []string{"linkedin.com", "licdn.com"}},
		{Name: "pinterest", DisplayName: "Pinterest", Category: CategorySocialNetworking,
			Hostnames: []string{"pinterest.com", "pinimg.com"}},
		{Name: "reddit", DisplayName: "Reddit", Category: CategorySocialNetworking,
			Hostnames: []string{"reddit.com", "redd.it", "redditstatic.com", "redditmedia.com"}},
		{Name: "snapchat", DisplayName: "Snapchat", Category: CategorySocialNetworking,
			Hostnames: []string{"snapchat.com", "snap.com", "snapkit.com", "sc-cdn.net"}},
		{Name: "tumblr", DisplayName: "Tumblr", Category: CategorySocialNetworking,
			Hostnames: []string{"tumblr.com"}},

		// Messaging
		{Name: "whatsapp", DisplayName: "WhatsApp", Category: CategoryMessaging,
			Hostnames: []string{"whatsapp.com", "whatsapp.net", "wa.me"}},
		{Name: "telegram", DisplayName: "Telegram", Category: CategoryMessaging,
			Hostnames: []string{"telegram.org", "telegram.me", "t.me", "tdesktop.com", "telesco.pe"}},
		{Name: "discord", DisplayName: "Discord", Category: CategoryMessaging,
			Hostnames: []string{"discord.com", "discord.gg", "discordapp.com", "discordapp.net", "discord.media"}},
		{Name: "slack", DisplayName: "Slack", Category: CategoryMessaging,
			Hostnames: []string{"slack.com", "slack-edge.com", "slack-msgs.com", "slack-imgs.com"}},
		{Name: "microsoft_teams", DisplayName: "Microsoft Teams", Category: CategoryMessaging,
			Hostnames: []string{"teams.microsoft.com", "teams.live.com", "teams.office.com"}},
		{Name: "zoom", DisplayName: "Zoom", Category: CategoryMessaging,
			Hostnames: []string{"zoom.us", "zoom.com", "zoomcdn.com"}},
		{Name: "skype", DisplayName: "Skype", Category: CategoryMessaging,
			Hostnames: []string{"skype.com", "skype.net", "skypeassets.com"}},
		{Name: "signal", DisplayName: "Signal", Category: CategoryMessaging,
			Hostnames: []string{"signal.org", "whispersystems.org"}},
		{Name: "viber", DisplayName: "Viber", Category: CategoryMessaging,
			Hostnames: []string{"viber.com"}},
		{Name: "line", DisplayName: "LINE", Category: CategoryMessaging,
			Hostnames: []string{"line.me", "line-scdn.net", "line-apps.com"}},

		// Games
		{Name: "steam", DisplayName: "Steam", Category: CategoryGames,
			Hostnames: []string{"steampowered.com", "steamcommunity.com", "steamgames.com", "steamstatic.com", "steamcontent.com"}},
		{Name: "epic_games", DisplayName: "Epic Games", Category: CategoryGames,
			Hostnames: []string{"epicgames.com", "unrealengine.com", "fortnite.com"}},
		{Name: "playstation", DisplayName: "PlayStation Network", Category: CategoryGames,
			Hostnames: []string{"playstation.com", "playstation.net", "sonyentertainmentnetwork.com", "sie.com"}},
		{Name: "xbox", DisplayName: "Xbox Live", Category: CategoryGames,
			Hostnames: []string{"xbox.com", "xboxlive.com", "xbox.net"}},
		{Name: "nintendo", DisplayName: "Nintendo Online", Category: CategoryGames,
			Hostnames: []string{"nintendo.com", "nintendo.net"}},
		{Name: "riot_games", DisplayName: "Riot Games", Category: CategoryGames,
			Hostnames: []string{"riotgames.com", "leagueoflegends.com"}},
		{Name: "blizzard", DisplayName: "Blizzard", Category: CategoryGames,
			Hostnames: []string{"blizzard.com", "battle.net", "blizzard.cn"}},
		{Name: "ea", DisplayName: "EA Games", Category: CategoryGames,
			Hostnames: []string{"ea.com", "origin.com"}},
		{Name: "ubisoft", DisplayName: "Ubisoft", Category: CategoryGames,
			Hostnames: []string{"ubisoft.com", "ubi.com"}},
		{Name: "roblox", DisplayName: "Roblox", Category: CategoryGames,
			Hostnames: []string{"roblox.com", "rbxcdn.com"}},

		// VPN/Tunneling
		{Name: "openvpn", DisplayName: "OpenVPN", Category: CategoryVPNTunneling,
			Hostnames: []string{"openvpn.net"}},
		{Name: "nordvpn", DisplayName: "NordVPN", Category: CategoryVPNTunneling,
			Hostnames: []string{"nordvpn.com", "nordcdn.com"}},
		{Name: "expressvpn", DisplayName: "ExpressVPN", Category: CategoryVPNTunneling,
			Hostnames: []string{"expressvpn.com", "xvpn.io"}},
		{Name: "surfshark", DisplayName: "Surfshark", Category: CategoryVPNTunneling,
			Hostnames: []string{"surfshark.com"}},
		{Name: "protonvpn", DisplayName: "ProtonVPN", Category: CategoryVPNTunneling,
			Hostnames: []string{"protonvpn.com", "proton.me"}},

		// File Transfer
		{Name: "dropbox", DisplayName: "Dropbox", Category: CategoryFileTransfer,
			Hostnames: []string{"dropbox.com", "dropboxapi.com", "dropboxstatic.com"}},
		{Name: "google_drive", DisplayName: "Google Drive", Category: CategoryFileTransfer,
			Hostnames: []string{"drive.google.com", "docs.google.com", "googleapis.com"}},
		{Name: "onedrive", DisplayName: "OneDrive", Category: CategoryFileTransfer,
			Hostnames: []string{"onedrive.live.com", "onedrive.com", "1drv.com", "1drv.ms"}},
		{Name: "icloud", DisplayName: "iCloud", Category: CategoryFileTransfer,
			Hostnames: []string{"icloud.com", "icloud-content.com"}},
		{Name: "wetransfer", DisplayName: "WeTransfer", Category: CategoryFileTransfer,
			Hostnames: []string{"wetransfer.com", "we.tl"}},
		{Name: "mega", DisplayName: "MEGA", Category: CategoryFileTransfer,
			Hostnames: []string{"mega.nz", "mega.co.nz", "mega.io"}},
		{Name: "mediafire", DisplayName: "MediaFire", Category: CategoryFileTransfer,
			Hostnames: []string{"mediafire.com"}},

		// Business
		{Name: "office365", DisplayName: "Microsoft 365", Category: CategoryBusiness,
			Hostnames: []string{"office.com", "office365.com", "microsoft365.com", "microsoftonline.com", "sharepoint.com", "outlook.com", "outlook.office.com"}},
		{Name: "google_workspace", DisplayName: "Google Workspace", Category: CategoryBusiness,
			Hostnames: []string{"google.com", "gmail.com", "googleusercontent.com"}},
		{Name: "salesforce", DisplayName: "Salesforce", Category: CategoryBusiness,
			Hostnames: []string{"salesforce.com", "force.com", "salesforceliveagent.com"}},
		{Name: "hubspot", DisplayName: "HubSpot", Category: CategoryBusiness,
			Hostnames: []string{"hubspot.com", "hubspotusercontent.com"}},
		{Name: "zendesk", DisplayName: "Zendesk", Category: CategoryBusiness,
			Hostnames: []string{"zendesk.com", "zdassets.com"}},
		{Name: "atlassian", DisplayName: "Atlassian", Category: CategoryBusiness,
			Hostnames: []string{"atlassian.com", "atlassian.net", "jira.com", "confluence.com", "trello.com", "bitbucket.org"}},
		{Name: "asana", DisplayName: "Asana", Category: CategoryBusiness,
			Hostnames: []string{"asana.com"}},
		{Name: "notion", DisplayName: "Notion", Category: CategoryBusiness,
			Hostnames: []string{"notion.so", "notion.com"}},

		// Ads (for blocking)
		{Name: "google_ads", DisplayName: "Google Ads", Category: CategoryAds,
			Hostnames: []string{"googleadservices.com", "googlesyndication.com", "doubleclick.net", "googleads.g.doubleclick.net", "adservice.google.com"}},
		{Name: "facebook_ads", DisplayName: "Facebook Ads", Category: CategoryAds,
			Hostnames: []string{"facebook.com/ads", "an.facebook.com"}},
	}

	for i := range apps {
		app := &apps[i]
		d.apps[app.Name] = app

		// Build hostname index
		for _, hostname := range app.Hostnames {
			d.hostnameIndex[hostname] = app.Name
		}
	}
}

// DetectByHostname detects the application based on the hostname (SNI).
func (d *Detector) DetectByHostname(hostname string) (*AppDefinition, float32) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	hostname = strings.ToLower(hostname)

	// Direct match
	if appName, ok := d.hostnameIndex[hostname]; ok {
		return d.apps[appName], 1.0
	}

	// Suffix match (e.g., api.netflix.com matches netflix.com)
	for suffix, appName := range d.hostnameIndex {
		if strings.HasSuffix(hostname, "."+suffix) || hostname == suffix {
			return d.apps[appName], 0.9
		}
	}

	return nil, 0
}

// GetApp returns an app definition by name.
func (d *Detector) GetApp(name string) *AppDefinition {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.apps[name]
}

// ListApps returns all known applications.
func (d *Detector) ListApps() []*AppDefinition {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]*AppDefinition, 0, len(d.apps))
	for _, app := range d.apps {
		result = append(result, app)
	}
	return result
}

// ListAppNames returns all known application names.
func (d *Detector) ListAppNames() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]string, 0, len(d.apps))
	for name := range d.apps {
		result = append(result, name)
	}
	return result
}

// ListAppsByCategory returns applications grouped by category.
func (d *Detector) ListAppsByCategory() map[AppCategory][]*AppDefinition {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[AppCategory][]*AppDefinition)
	for _, app := range d.apps {
		result[app.Category] = append(result[app.Category], app)
	}
	return result
}

// AddCustomApp adds a custom application definition.
func (d *Detector) AddCustomApp(app *AppDefinition) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.apps[app.Name] = app
	for _, hostname := range app.Hostnames {
		d.hostnameIndex[hostname] = app.Name
	}
}
