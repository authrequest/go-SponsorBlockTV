package constants

const (
	// UserAgent is the user agent string for API requests
	UserAgent = "go-SponsorBlockTV/0.1"

	// SponsorBlockService is the service name for SponsorBlock
	SponsorBlockService = "youtube"

	// SponsorBlockActionType is the action type for SponsorBlock
	SponsorBlockActionType = "skip"

	// SponsorBlockAPI is the base URL for the SponsorBlock API
	SponsorBlockAPI = "https://sponsor.ajay.app/api"

	// YouTube API constants
	YouTubeAPI = "https://www.googleapis.com/youtube/v3"

	// GitHub constants
	GitHubWikiBaseURL = "https://github.com/dmunozv04/iSponsorBlockTV/wiki"
)

// SkipCategory represents a category of segments to skip
type SkipCategory struct {
	Name string
	ID   string
}

// SkipCategories is a list of sponsor categories that can be skipped
var SkipCategories = []SkipCategory{
	{"Sponsor", "sponsor"},
	{"Self Promotion", "selfpromo"},
	{"Intro", "intro"},
	{"Outro", "outro"},
	{"Music Offtopic", "music_offtopic"},
	{"Interaction", "interaction"},
	{"Exclusive Access", "exclusive_access"},
	{"POI Highlight", "poi_highlight"},
	{"Preview", "preview"},
	{"Filler", "filler"},
}

// YouTubeClientBlacklist is a list of YouTube clients that should be blacklisted
var YouTubeClientBlacklist = []string{"TVHTML5_FOR_KIDS"}

// ConfigFileBlacklistKeys is a list of keys that should not be saved to the config file
var ConfigFileBlacklistKeys = []string{"config_file", "data_dir"}

// GetSkipCategoryIDs returns a slice of all skip category IDs
func GetSkipCategoryIDs() []string {
	ids := make([]string, len(SkipCategories))
	for i, category := range SkipCategories {
		ids[i] = category.ID
	}
	return ids
}

// GetSkipCategoryByID returns the skip category with the given ID
func GetSkipCategoryByID(id string) (SkipCategory, bool) {
	for _, category := range SkipCategories {
		if category.ID == id {
			return category, true
		}
	}
	return SkipCategory{}, false
}

// GetSkipCategoryByName returns the skip category with the given name
func GetSkipCategoryByName(name string) (SkipCategory, bool) {
	for _, category := range SkipCategories {
		if category.Name == name {
			return category, true
		}
	}
	return SkipCategory{}, false
}
