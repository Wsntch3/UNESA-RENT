package catalog

import (
	"context"
	"time"
)

type Store interface {
	ListCategories(ctx context.Context) ([]Category, error)
	ListLocations(ctx context.Context) ([]Location, error)

	IsVerifiedOwner(ctx context.Context, userID string) (bool, error)

	Create(ctx context.Context, ownerID string, in ListingInput) (Listing, error)
	Update(ctx context.Context, id string, in ListingInput) (Listing, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (Listing, error)
	Search(ctx context.Context, f Filter) ([]Listing, int, error)
	ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]Listing, int, error)

	AddPhoto(ctx context.Context, listingID, storageKey, mimeType string, size int64) (Photo, error)
	DeletePhoto(ctx context.Context, listingID, photoID string) (Photo, error)
	GetPhoto(ctx context.Context, listingID, photoID string) (Photo, error)
	CountPhotos(ctx context.Context, listingID string) (int, error)
	ListPhotos(ctx context.Context, listingID string) ([]Photo, error)

	AddBlock(ctx context.Context, listingID string, start, end time.Time, reason *string) (Block, error)
	ListBlocks(ctx context.Context, listingID string) ([]Block, error)
	DeleteBlock(ctx context.Context, listingID, blockID string) error

	AdminList(ctx context.Context, status, query string, limit, offset int) ([]Listing, int, error)
	Takedown(ctx context.Context, id, adminID, reason string) error
	Restore(ctx context.Context, id string) error
	LogAction(ctx context.Context, actorID, action, targetID, reason string) error
}
