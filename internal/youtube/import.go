package youtube

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/youtube/v3"
	"ytmigrator/internal/state"
)

// ImportSubscriptions copies subscriptions from the bundle to the target account.
func (c *Client) ImportSubscriptions(ctx context.Context, bundle *ExportBundle, prog *state.ImportProgress) error {
	log.Printf("Importing %d subscriptions...", len(bundle.Subscriptions))
	for i, sub := range bundle.Subscriptions {
		err := c.importSubscription(ctx, sub)
		if err != nil {
			log.Printf("  subscription %d/%d FAILED: %s - %v", i+1, len(bundle.Subscriptions), sub.Title, err)
			prog.MarkSubscriptionDone(sub.ChannelId, false)
		} else {
			log.Printf("  subscription %d/%d OK: %s", i+1, len(bundle.Subscriptions), sub.Title)
			prog.MarkSubscriptionDone(sub.ChannelId, true)
		}
		prog.AddQuota(51)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	log.Println("Subscriptions import done")
	return nil
}

func (c *Client) importSubscription(ctx context.Context, sub Subscription) error {
	item := &youtube.Subscription{
		Snippet: &youtube.SubscriptionSnippet{
			ResourceId: &youtube.ResourceId{
				Kind:      "youtube#channel",
				ChannelId: sub.ChannelId,
			},
		},
	}
	// Map notification level (ActivityType in v3 API)
	if sub.NotificationLevel != "" {
		item.ContentDetails = &youtube.SubscriptionContentDetails{
			ActivityType: sub.NotificationLevel,
		}
	}

	_, err := c.service.Subscriptions.Insert([]string{"snippet", "contentDetails"}, item).Context(ctx).Do()
	if err != nil {
		if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code == 409 {
			return nil // already subscribed
		}
		return fmt.Errorf("subscribe to %s: %w", sub.ChannelId, err)
	}
	return nil
}

// ImportPlaylists recreates playlists and populates them with items.
func (c *Client) ImportPlaylists(ctx context.Context, bundle *ExportBundle, prog *state.ImportProgress) error {
	for _, pl := range bundle.Playlists {
		newID, err := c.importPlaylist(ctx, pl)
		if err != nil {
			prog.MarkPlaylistDone(pl.ID, false)
			continue
		}
		prog.MarkPlaylistDone(pl.ID, true)
		prog.AddQuota(52) // list(1) + insert(1) + items

		// Add items
		for _, item := range pl.Items {
			err := c.importPlaylistItem(ctx, newID, item)
			if err != nil {
				// item-level failure not globally tracked; continue
			}
			prog.AddQuota(50)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return nil
}

func (c *Client) importPlaylist(ctx context.Context, pl Playlist) (string, error) {
	item := &youtube.Playlist{
		Snippet: &youtube.PlaylistSnippet{
			Title:       pl.Title,
			Description: pl.Description,
		},
		Status: &youtube.PlaylistStatus{
			PrivacyStatus: pl.Privacy,
		},
	}
	resp, err := c.service.Playlists.Insert([]string{"snippet", "status"}, item).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("insert playlist %s: %w", pl.Title, err)
	}
	return resp.Id, nil
}

func (c *Client) importPlaylistItem(ctx context.Context, playlistID string, item PlaylistItem) error {
	pi := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: item.VideoID,
			},
		},
	}
	_, err := c.service.PlaylistItems.Insert([]string{"snippet"}, pi).Context(ctx).Do()
	if err != nil {
		if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code == 409 {
			return nil // already in playlist
		}
		return fmt.Errorf("add item %s to playlist %s: %w", item.VideoID, playlistID, err)
	}
	return nil
}

// ImportLikes rates videos "like" in the target account.
func (c *Client) ImportLikes(ctx context.Context, bundle *ExportBundle, prog *state.ImportProgress) error {
	for _, vid := range bundle.LikedVideos {
		err := c.service.Videos.Rate(vid.VideoID, "like").Context(ctx).Do()
		if err != nil {
			prog.MarkLikedVideoDone(vid.VideoID, false)
		} else {
			prog.MarkLikedVideoDone(vid.VideoID, true)
		}
		prog.AddQuota(50)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return nil
}

// ImportAll runs all import phases in sequence.
func (c *Client) ImportAll(ctx context.Context, bundle *ExportBundle, prog *state.ImportProgress) error {
	if err := c.ImportSubscriptions(ctx, bundle, prog); err != nil {
		return fmt.Errorf("subscriptions: %w", err)
	}
	if err := c.ImportPlaylists(ctx, bundle, prog); err != nil {
		return fmt.Errorf("playlists: %w", err)
	}
	if err := c.ImportLikes(ctx, bundle, prog); err != nil {
		return fmt.Errorf("likes: %w", err)
	}
	return nil
}
