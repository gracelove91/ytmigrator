// Package youtube wraps the YouTube Data API v3 for export/import operations.
package youtube

import (
	"context"
	"fmt"
	"net/http"
	"time"

	yt "google.golang.org/api/youtube/v3"
)

// Client wraps the YouTube service with an authenticated HTTP client.
type Client struct {
	service *yt.Service
}

// NewClient creates a new Client from an existing access token.
// The token will auto-refresh if a refresh token was included in the original exchange.
func NewClient(ctx context.Context, token *http.Client) (*Client, error) {
	svc, err := yt.New(token)
	if err != nil {
		return nil, fmt.Errorf("create youtube service: %w", err)
	}
	return &Client{service: svc}, nil
}

// ExportSubscriptions returns all channel subscriptions of the authenticated user.
func (c *Client) ExportSubscriptions(ctx context.Context) ([]Subscription, error) {
	var result []Subscription
	var pageToken string

	for {
		call := c.service.Subscriptions.List([]string{"snippet", "contentDetails"}).
			Mine(true).
			MaxResults(50)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("subscriptions.list: %w", err)
		}

		for _, item := range resp.Items {
		sub := Subscription{
			ChannelId:         item.Snippet.ResourceId.ChannelId,
			Title:             item.Snippet.Title,
			NotificationLevel: item.ContentDetails.ActivityType,
		}
			result = append(result, sub)
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return result, nil
}

// ExportPlaylists returns all playlists (including Watch Later which has a special ID).
func (c *Client) ExportPlaylists(ctx context.Context) ([]Playlist, error) {
	var playlists []Playlist
	var pageToken string

	// Step 1: list playlists
	for {
		call := c.service.Playlists.List([]string{"snippet", "status"}).
			Mine(true).
			MaxResults(50)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("playlists.list: %w", err)
		}

		for _, pl := range resp.Items {
			playlists = append(playlists, Playlist{
				ID:          pl.Id,
				Title:       pl.Snippet.Title,
				Description: pl.Snippet.Description,
				Privacy:     pl.Status.PrivacyStatus,
			})
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	// Step 2: fetch items for each playlist
	for i := range playlists {
		items, err := c.exportPlaylistItems(ctx, playlists[i].ID)
		if err != nil {
			return nil, fmt.Errorf("playlistItems.list for %s: %w", playlists[i].ID, err)
		}
		playlists[i].Items = items
	}

	return playlists, nil
}

func (c *Client) exportPlaylistItems(ctx context.Context, playlistID string) ([]PlaylistItem, error) {
	var result []PlaylistItem
	var pageToken string

	for {
		call := c.service.PlaylistItems.List([]string{"snippet"}).
			PlaylistId(playlistID).
			MaxResults(50)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			result = append(result, PlaylistItem{
				VideoID: item.Snippet.ResourceId.VideoId,
				Title:   item.Snippet.Title,
			})
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return result, nil
}

// ExportLikedVideos returns all videos the user rated "like".
func (c *Client) ExportLikedVideos(ctx context.Context) ([]LikedVideo, error) {
	var result []LikedVideo
	var pageToken string

	for {
		call := c.service.Videos.List([]string{"snippet"}).
			MyRating("like").
			MaxResults(50)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("videos.list myRating=like: %w", err)
		}

		for _, vid := range resp.Items {
			result = append(result, LikedVideo{
				VideoID: vid.Id,
				Title:   vid.Snippet.Title,
			})
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return result, nil
}

// ExportAll exports subscriptions, playlists, and liked videos into a bundle.
func (c *Client) ExportAll(ctx context.Context) (*ExportBundle, error) {
	subs, err := c.ExportSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("export subscriptions: %w", err)
	}

	playlists, err := c.ExportPlaylists(ctx)
	if err != nil {
		return nil, fmt.Errorf("export playlists: %w", err)
	}

	liked, err := c.ExportLikedVideos(ctx)
	if err != nil {
		return nil, fmt.Errorf("export liked videos: %w", err)
	}

	return &ExportBundle{
		Version:       1,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Subscriptions: subs,
		Playlists:     playlists,
		LikedVideos:   liked,
	}, nil
}
