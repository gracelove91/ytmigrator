// Package youtube wraps the YouTube Data API v3 for export/import operations.
package youtube

// Subscription represents a YouTube channel subscription.
type Subscription struct {
	ChannelId         string `json:"channelId"`
	Title             string `json:"title"`
	NotificationLevel string `json:"notificationLevel"` // "all", "occasional", "none"
}

// PlaylistItem is a single video in a playlist.
type PlaylistItem struct {
	VideoID string `json:"videoId"`
	Title   string `json:"title"`
}

// Playlist represents a user-created playlist including items.
type Playlist struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Privacy     string         `json:"privacy"` // public | private | unlisted
	Items       []PlaylistItem `json:"items"`
}

// LikedVideo represents a video the user rated "like".
type LikedVideo struct {
	VideoID string `json:"videoId"`
	Title   string `json:"title"`
}

// ExportBundle is the standard JSON envelope for exported data.
type ExportBundle struct {
	Version       int            `json:"version"`
	ExportedAt    string         `json:"exportedAt"`
	Subscriptions []Subscription `json:"subscriptions"`
	Playlists     []Playlist     `json:"playlists"`
	LikedVideos   []LikedVideo   `json:"likedVideos"`
}
