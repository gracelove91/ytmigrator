package state

import (
	"encoding/json"
	"os"
	"sync"
)

// ImportProgress tracks which items have been migrated and quota consumption.
type ImportProgress struct {
	mu sync.Mutex `json:"-"`

	SubscriptionsCompleted []string `json:"subscriptions_completed"`
	SubscriptionsFailed    []string `json:"subscriptions_failed"`
	PlaylistsCompleted     []string `json:"playlists_completed"`
	PlaylistsFailed        []string `json:"playlists_failed"`
	LikedVideosCompleted   []string `json:"liked_videos_completed"`
	LikedVideosFailed      []string `json:"liked_videos_failed"`
	QuotaUsed              int      `json:"quota_used"`
}

func (p *ImportProgress) AddQuota(n int) {
	p.mu.Lock()
	p.QuotaUsed += n
	p.mu.Unlock()
}

func (p *ImportProgress) IsSubscriptionDone(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return contains(p.SubscriptionsCompleted, id) || contains(p.SubscriptionsFailed, id)
}

func (p *ImportProgress) MarkSubscriptionDone(id string, success bool) {
	p.mu.Lock()
	if success {
		p.SubscriptionsCompleted = append(p.SubscriptionsCompleted, id)
	} else {
		p.SubscriptionsFailed = append(p.SubscriptionsFailed, id)
	}
	p.mu.Unlock()
}

func (p *ImportProgress) IsPlaylistDone(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return contains(p.PlaylistsCompleted, id) || contains(p.PlaylistsFailed, id)
}

func (p *ImportProgress) MarkPlaylistDone(id string, success bool) {
	p.mu.Lock()
	if success {
		p.PlaylistsCompleted = append(p.PlaylistsCompleted, id)
	} else {
		p.PlaylistsFailed = append(p.PlaylistsFailed, id)
	}
	p.mu.Unlock()
}

func (p *ImportProgress) IsLikedVideoDone(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return contains(p.LikedVideosCompleted, id) || contains(p.LikedVideosFailed, id)
}

func (p *ImportProgress) MarkLikedVideoDone(id string, success bool) {
	p.mu.Lock()
	if success {
		p.LikedVideosCompleted = append(p.LikedVideosCompleted, id)
	} else {
		p.LikedVideosFailed = append(p.LikedVideosFailed, id)
	}
	p.mu.Unlock()
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// LoadImportProgress reads progress from disk. Returns an empty progress if not found.
func LoadImportProgress(path string) (*ImportProgress, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ImportProgress{}, nil
		}
		return nil, err
	}
	var p ImportProgress
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SaveImportProgress writes current progress to disk.
func SaveImportProgress(path string, p *ImportProgress) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
