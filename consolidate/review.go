package consolidate

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/plexusone/signal-spec/pkg/rootcause"
)

// Review status constants.
const (
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

// Common review errors.
var (
	ErrReviewNotFound   = errors.New("review not found")
	ErrAlreadyReviewed  = errors.New("root cause already reviewed")
	ErrReviewerRequired = errors.New("reviewer not configured")
)

// ReviewItem represents a root cause pending review.
type ReviewItem struct {
	RootCause   rootcause.RootCause
	SubmittedAt time.Time
	ReviewedAt  *time.Time
	ReviewedBy  string
	Status      string
	Notes       string
}

// ReviewHook is called when review events occur.
type ReviewHook func(ctx context.Context, item ReviewItem) error

// ReviewConfig configures the review system.
type ReviewConfig struct {
	// RequireReview blocks root causes until approved.
	// If false, unreviewed root causes are flagged but not blocked.
	RequireReview bool

	// OnSubmit is called when a root cause is submitted for review.
	OnSubmit ReviewHook

	// OnApprove is called when a root cause is approved.
	OnApprove ReviewHook

	// OnReject is called when a root cause is rejected.
	OnReject ReviewHook

	// AutoApproveThreshold auto-approves if signal count exceeds this.
	// Zero disables auto-approve.
	AutoApproveThreshold int
}

// MemoryReviewer is an in-memory implementation of the Reviewer interface.
type MemoryReviewer struct {
	mu     sync.RWMutex
	items  map[string]*ReviewItem
	config ReviewConfig
}

// NewMemoryReviewer creates an in-memory review queue.
func NewMemoryReviewer(cfg ReviewConfig) *MemoryReviewer {
	return &MemoryReviewer{
		items:  make(map[string]*ReviewItem),
		config: cfg,
	}
}

// Submit queues a root cause for review.
func (r *MemoryReviewer) Submit(ctx context.Context, rc rootcause.RootCause) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item := &ReviewItem{
		RootCause:   rc,
		SubmittedAt: time.Now(),
		Status:      ReviewPending,
	}

	// Check for auto-approve
	if r.config.AutoApproveThreshold > 0 && len(rc.SignalIDs) >= r.config.AutoApproveThreshold {
		now := time.Now()
		item.Status = ReviewApproved
		item.ReviewedAt = &now
		item.ReviewedBy = "auto"
		item.Notes = "Auto-approved: signal count threshold met"
	}

	r.items[rc.ID] = item

	if r.config.OnSubmit != nil {
		if err := r.config.OnSubmit(ctx, *item); err != nil {
			return err
		}
	}

	return nil
}

// Pending returns root causes awaiting review.
func (r *MemoryReviewer) Pending(ctx context.Context) ([]rootcause.RootCause, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []rootcause.RootCause
	for _, item := range r.items {
		if item.Status == ReviewPending {
			result = append(result, item.RootCause)
		}
	}
	return result, nil
}

// Approve marks a root cause as reviewed and approved.
func (r *MemoryReviewer) Approve(ctx context.Context, id string) error {
	return r.ApproveWithDetails(ctx, id, "", "")
}

// ApproveWithDetails approves with reviewer info and notes.
func (r *MemoryReviewer) ApproveWithDetails(ctx context.Context, id, reviewer, notes string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return ErrReviewNotFound
	}

	if item.Status != ReviewPending {
		return ErrAlreadyReviewed
	}

	now := time.Now()
	item.Status = ReviewApproved
	item.ReviewedAt = &now
	item.ReviewedBy = reviewer
	item.Notes = notes

	if r.config.OnApprove != nil {
		if err := r.config.OnApprove(ctx, *item); err != nil {
			return err
		}
	}

	return nil
}

// Reject marks a root cause as reviewed and rejected.
func (r *MemoryReviewer) Reject(ctx context.Context, id string, reason string) error {
	return r.RejectWithDetails(ctx, id, "", reason)
}

// RejectWithDetails rejects with reviewer info.
func (r *MemoryReviewer) RejectWithDetails(ctx context.Context, id, reviewer, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return ErrReviewNotFound
	}

	if item.Status != ReviewPending {
		return ErrAlreadyReviewed
	}

	now := time.Now()
	item.Status = ReviewRejected
	item.ReviewedAt = &now
	item.ReviewedBy = reviewer
	item.Notes = reason

	if r.config.OnReject != nil {
		if err := r.config.OnReject(ctx, *item); err != nil {
			return err
		}
	}

	return nil
}

// Get retrieves a review item by root cause ID.
func (r *MemoryReviewer) Get(ctx context.Context, id string) (*ReviewItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return nil, ErrReviewNotFound
	}
	return item, nil
}

// List returns all review items.
func (r *MemoryReviewer) List(ctx context.Context) ([]ReviewItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ReviewItem, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, *item)
	}
	return result, nil
}

// ListByStatus returns review items with the specified status.
func (r *MemoryReviewer) ListByStatus(ctx context.Context, status string) ([]ReviewItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ReviewItem
	for _, item := range r.items {
		if item.Status == status {
			result = append(result, *item)
		}
	}
	return result, nil
}

// PendingCount returns the number of items awaiting review.
func (r *MemoryReviewer) PendingCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, item := range r.items {
		if item.Status == ReviewPending {
			count++
		}
	}
	return count
}

// IsApproved checks if a root cause has been approved.
func (r *MemoryReviewer) IsApproved(ctx context.Context, id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return false, nil
	}
	return item.Status == ReviewApproved, nil
}

// RequiresReview returns true if root causes must be approved before use.
func (r *MemoryReviewer) RequiresReview() bool {
	return r.config.RequireReview
}

// Clear removes all review items (for testing).
func (r *MemoryReviewer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]*ReviewItem)
}
