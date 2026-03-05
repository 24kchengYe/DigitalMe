package licensing

import "fmt"

const watermarkInterval = 5 // append watermark every N messages
const watermarkText = "\n\n— DigitalMe"

// ShouldWatermark returns true every watermarkInterval messages.
func ShouldWatermark() bool {
	count := IncrementMessageCount()
	return count%watermarkInterval == 0
}

// ApplyWatermark conditionally appends the DigitalMe watermark to content.
func ApplyWatermark(content string) string {
	if ShouldWatermark() {
		return content + watermarkText
	}
	return content
}

// StartupBanner returns an ASCII art banner with license status.
func StartupBanner() string {
	tier := GetTier()
	status := StatusString()

	banner := `
 ____  _       _ _        _ __  __
|  _ \(_) __ _(_) |_ __ _| |  \/  | ___
| | | | |/ _` + "`" + ` | | __/ _` + "`" + ` | | |\/| |/ _ \
| |_| | | (_| | | || (_| | | |  | |  __/
|____/|_|\__, |_|\__\__,_|_|_|  |_|\___|
         |___/
`
	if tier == TierPro {
		banner += fmt.Sprintf("  [Pro] %s\n", status)
	} else {
		banner += fmt.Sprintf("  [Free] %s\n", status)
	}

	return banner
}

// DashboardFooter returns the branding text for the web dashboard footer.
func DashboardFooter() string {
	return "Powered by DigitalMe"
}

// DashboardLicenseInfo returns license info for dashboard display.
func DashboardLicenseInfo() string {
	tier := GetTier()
	payload := GetPayload()

	if payload == nil {
		return "Free Tier"
	}

	info := fmt.Sprintf("%s | %s", payload.Licensee, string(tier))
	if payload.Expires != "" {
		if payload.IsExpired() {
			info += " | Expired"
		} else {
			info += fmt.Sprintf(" | Until %s", payload.Expires)
		}
	}
	return info
}
