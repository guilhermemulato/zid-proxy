package appid

import (
	"testing"
)

func TestDetector_DetectByHostname(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		hostname    string
		wantApp     string
		wantFound   bool
		minConfidence float32
	}{
		// Exact matches
		{"netflix.com", "netflix", true, 0.9},
		{"youtube.com", "youtube", true, 0.9},
		{"facebook.com", "facebook", true, 0.9},
		{"whatsapp.com", "whatsapp", true, 0.9},

		// Subdomain matches
		{"api.netflix.com", "netflix", true, 0.9},
		{"www.youtube.com", "youtube", true, 0.9},
		{"m.facebook.com", "facebook", true, 0.9},

		// CDN domains
		{"nflxvideo.net", "netflix", true, 0.9},
		{"googlevideo.com", "youtube", true, 0.9},
		{"fbcdn.net", "facebook", true, 0.9},
		{"cdninstagram.com", "instagram", true, 0.9},

		// TikTok
		{"tiktok.com", "tiktok", true, 0.9},
		{"tiktokcdn.com", "tiktok", true, 0.9},

		// Games
		{"steampowered.com", "steam", true, 0.9},
		{"epicgames.com", "epic_games", true, 0.9},

		// VPN
		{"nordvpn.com", "nordvpn", true, 0.9},

		// No match
		{"example.com", "", false, 0},
		{"unknown-domain.net", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			app, confidence := d.DetectByHostname(tt.hostname)

			if tt.wantFound {
				if app == nil {
					t.Errorf("expected to find app for %s, got nil", tt.hostname)
					return
				}
				if app.Name != tt.wantApp {
					t.Errorf("expected app %s, got %s", tt.wantApp, app.Name)
				}
				if confidence < tt.minConfidence {
					t.Errorf("expected confidence >= %.2f, got %.2f", tt.minConfidence, confidence)
				}
			} else {
				if app != nil {
					t.Errorf("expected no match for %s, got %s", tt.hostname, app.Name)
				}
			}
		})
	}
}

func TestDetector_ListApps(t *testing.T) {
	d := NewDetector()
	apps := d.ListApps()

	if len(apps) == 0 {
		t.Error("expected at least some apps, got none")
	}

	// Check that important apps are present
	names := make(map[string]bool)
	for _, app := range apps {
		names[app.Name] = true
	}

	mustHave := []string{"netflix", "youtube", "facebook", "whatsapp", "telegram", "steam"}
	for _, name := range mustHave {
		if !names[name] {
			t.Errorf("expected app %s to be in the list", name)
		}
	}
}

func TestDetector_ListAppsByCategory(t *testing.T) {
	d := NewDetector()
	byCategory := d.ListAppsByCategory()

	// Check that main categories have apps
	expectedCategories := []AppCategory{
		CategoryStreamingMedia,
		CategorySocialNetworking,
		CategoryMessaging,
		CategoryGames,
	}

	for _, cat := range expectedCategories {
		apps := byCategory[cat]
		if len(apps) == 0 {
			t.Errorf("expected apps in category %s, got none", cat)
		}
	}
}

func TestDetector_AddCustomApp(t *testing.T) {
	d := NewDetector()

	customApp := &AppDefinition{
		Name:        "custom_app",
		DisplayName: "Custom App",
		Category:    CategoryBusiness,
		Hostnames:   []string{"customapp.com", "api.customapp.com"},
	}

	d.AddCustomApp(customApp)

	// Test detection
	app, confidence := d.DetectByHostname("customapp.com")
	if app == nil || app.Name != "custom_app" {
		t.Error("expected to detect custom app")
	}
	if confidence < 0.9 {
		t.Errorf("expected high confidence, got %.2f", confidence)
	}

	// Test subdomain
	app, _ = d.DetectByHostname("www.customapp.com")
	if app == nil || app.Name != "custom_app" {
		t.Error("expected to detect custom app via subdomain")
	}
}
