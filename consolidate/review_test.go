package consolidate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/plexusone/omnisignal/consolidate"
	"github.com/plexusone/signal-spec/pkg/rootcause"
)

func TestNewMemoryReviewer(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	if r == nil {
		t.Fatal("NewMemoryReviewer returned nil")
	}
}

func TestMemoryReviewerSubmit(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	rc := rootcause.RootCause{ID: "rc-1", Title: "Test"}

	err := r.Submit(ctx, rc)
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}

	pending, err := r.Pending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("Pending = %d, want 1", len(pending))
	}
	if pending[0].ID != "rc-1" {
		t.Error("wrong ID in pending")
	}
}

func TestMemoryReviewerApprove(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	rc := rootcause.RootCause{ID: "rc-1"}
	_ = r.Submit(ctx, rc)

	err := r.Approve(ctx, "rc-1")
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}

	pending, _ := r.Pending(ctx)
	if len(pending) != 0 {
		t.Error("should have no pending after approve")
	}

	approved, err := r.IsApproved(ctx, "rc-1")
	if err != nil {
		t.Fatal(err)
	}
	if !approved {
		t.Error("should be approved")
	}
}

func TestMemoryReviewerApproveWithDetails(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	rc := rootcause.RootCause{ID: "rc-1"}
	_ = r.Submit(ctx, rc)

	err := r.ApproveWithDetails(ctx, "rc-1", "reviewer@example.com", "LGTM")
	if err != nil {
		t.Fatalf("ApproveWithDetails error: %v", err)
	}

	item, err := r.Get(ctx, "rc-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.ReviewedBy != "reviewer@example.com" {
		t.Errorf("ReviewedBy = %s", item.ReviewedBy)
	}
	if item.Notes != "LGTM" {
		t.Errorf("Notes = %s", item.Notes)
	}
	if item.ReviewedAt == nil {
		t.Error("ReviewedAt should be set")
	}
}

func TestMemoryReviewerReject(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	rc := rootcause.RootCause{ID: "rc-1"}
	_ = r.Submit(ctx, rc)

	err := r.Reject(ctx, "rc-1", "Duplicate of existing issue")
	if err != nil {
		t.Fatalf("Reject error: %v", err)
	}

	approved, _ := r.IsApproved(ctx, "rc-1")
	if approved {
		t.Error("should not be approved after rejection")
	}

	item, _ := r.Get(ctx, "rc-1")
	if item.Status != consolidate.ReviewRejected {
		t.Errorf("Status = %s, want rejected", item.Status)
	}
	if item.Notes != "Duplicate of existing issue" {
		t.Errorf("Notes = %s", item.Notes)
	}
}

func TestMemoryReviewerNotFound(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	err := r.Approve(ctx, "nonexistent")
	if !errors.Is(err, consolidate.ErrReviewNotFound) {
		t.Errorf("expected ErrReviewNotFound, got %v", err)
	}

	err = r.Reject(ctx, "nonexistent", "reason")
	if !errors.Is(err, consolidate.ErrReviewNotFound) {
		t.Errorf("expected ErrReviewNotFound, got %v", err)
	}

	_, err = r.Get(ctx, "nonexistent")
	if !errors.Is(err, consolidate.ErrReviewNotFound) {
		t.Errorf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestMemoryReviewerAlreadyReviewed(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	rc := rootcause.RootCause{ID: "rc-1"}
	_ = r.Submit(ctx, rc)
	_ = r.Approve(ctx, "rc-1")

	err := r.Approve(ctx, "rc-1")
	if !errors.Is(err, consolidate.ErrAlreadyReviewed) {
		t.Errorf("expected ErrAlreadyReviewed, got %v", err)
	}

	err = r.Reject(ctx, "rc-1", "reason")
	if !errors.Is(err, consolidate.ErrAlreadyReviewed) {
		t.Errorf("expected ErrAlreadyReviewed, got %v", err)
	}
}

func TestMemoryReviewerAutoApprove(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{
		AutoApproveThreshold: 5,
	})
	ctx := context.Background()

	rc := rootcause.RootCause{
		ID:        "rc-1",
		SignalIDs: []string{"s1", "s2", "s3", "s4", "s5", "s6"},
	}

	err := r.Submit(ctx, rc)
	if err != nil {
		t.Fatal(err)
	}

	approved, _ := r.IsApproved(ctx, "rc-1")
	if !approved {
		t.Error("should be auto-approved (6 signals >= threshold 5)")
	}

	item, _ := r.Get(ctx, "rc-1")
	if item.ReviewedBy != "auto" {
		t.Errorf("ReviewedBy = %s, want 'auto'", item.ReviewedBy)
	}
}

func TestMemoryReviewerAutoApproveNotTriggered(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{
		AutoApproveThreshold: 5,
	})
	ctx := context.Background()

	rc := rootcause.RootCause{
		ID:        "rc-1",
		SignalIDs: []string{"s1", "s2"},
	}

	_ = r.Submit(ctx, rc)

	approved, _ := r.IsApproved(ctx, "rc-1")
	if approved {
		t.Error("should not be auto-approved (2 signals < threshold 5)")
	}
}

func TestMemoryReviewerHooks(t *testing.T) {
	var submitCalled, approveCalled, rejectCalled bool

	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{
		OnSubmit: func(ctx context.Context, item consolidate.ReviewItem) error {
			submitCalled = true
			return nil
		},
		OnApprove: func(ctx context.Context, item consolidate.ReviewItem) error {
			approveCalled = true
			return nil
		},
		OnReject: func(ctx context.Context, item consolidate.ReviewItem) error {
			rejectCalled = true
			return nil
		},
	})
	ctx := context.Background()

	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-1"})
	if !submitCalled {
		t.Error("OnSubmit hook not called")
	}

	_ = r.Approve(ctx, "rc-1")
	if !approveCalled {
		t.Error("OnApprove hook not called")
	}

	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-2"})
	_ = r.Reject(ctx, "rc-2", "reason")
	if !rejectCalled {
		t.Error("OnReject hook not called")
	}
}

func TestMemoryReviewerHookError(t *testing.T) {
	expectedErr := errors.New("hook failed")
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{
		OnSubmit: func(ctx context.Context, item consolidate.ReviewItem) error {
			return expectedErr
		},
	})
	ctx := context.Background()

	err := r.Submit(ctx, rootcause.RootCause{ID: "rc-1"})
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected hook error, got %v", err)
	}
}

func TestMemoryReviewerList(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-1"})
	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-2"})
	_ = r.Approve(ctx, "rc-1")

	all, err := r.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("List returned %d, want 2", len(all))
	}

	pending, err := r.ListByStatus(ctx, consolidate.ReviewPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("Pending = %d, want 1", len(pending))
	}

	approved, err := r.ListByStatus(ctx, consolidate.ReviewApproved)
	if err != nil {
		t.Fatal(err)
	}
	if len(approved) != 1 {
		t.Errorf("Approved = %d, want 1", len(approved))
	}
}

func TestMemoryReviewerPendingCount(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	if r.PendingCount() != 0 {
		t.Error("initial pending count should be 0")
	}

	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-1"})
	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-2"})

	if r.PendingCount() != 2 {
		t.Errorf("PendingCount = %d, want 2", r.PendingCount())
	}

	_ = r.Approve(ctx, "rc-1")

	if r.PendingCount() != 1 {
		t.Errorf("PendingCount = %d, want 1", r.PendingCount())
	}
}

func TestMemoryReviewerRequiresReview(t *testing.T) {
	r1 := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{RequireReview: false})
	if r1.RequiresReview() {
		t.Error("should not require review")
	}

	r2 := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{RequireReview: true})
	if !r2.RequiresReview() {
		t.Error("should require review")
	}
}

func TestMemoryReviewerClear(t *testing.T) {
	r := consolidate.NewMemoryReviewer(consolidate.ReviewConfig{})
	ctx := context.Background()

	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-1"})
	_ = r.Submit(ctx, rootcause.RootCause{ID: "rc-2"})

	r.Clear()

	all, _ := r.List(ctx)
	if len(all) != 0 {
		t.Error("Clear should remove all items")
	}
}

func TestMemoryReviewerImplementsInterface(t *testing.T) {
	var _ consolidate.Reviewer = (*consolidate.MemoryReviewer)(nil)
}

func TestReviewItemFields(t *testing.T) {
	now := time.Now()
	item := consolidate.ReviewItem{
		RootCause:   rootcause.RootCause{ID: "rc-1"},
		SubmittedAt: now,
		ReviewedAt:  &now,
		ReviewedBy:  "reviewer",
		Status:      consolidate.ReviewApproved,
		Notes:       "Approved with notes",
	}

	if item.RootCause.ID != "rc-1" {
		t.Error("RootCause mismatch")
	}
	if item.SubmittedAt != now {
		t.Error("SubmittedAt mismatch")
	}
	if item.ReviewedAt == nil || *item.ReviewedAt != now {
		t.Error("ReviewedAt mismatch")
	}
	if item.ReviewedBy != "reviewer" {
		t.Error("ReviewedBy mismatch")
	}
	if item.Status != consolidate.ReviewApproved {
		t.Error("Status mismatch")
	}
	if item.Notes != "Approved with notes" {
		t.Error("Notes mismatch")
	}
}

func TestReviewStatusConstants(t *testing.T) {
	if consolidate.ReviewPending != "pending" {
		t.Error("ReviewPending mismatch")
	}
	if consolidate.ReviewApproved != "approved" {
		t.Error("ReviewApproved mismatch")
	}
	if consolidate.ReviewRejected != "rejected" {
		t.Error("ReviewRejected mismatch")
	}
}
