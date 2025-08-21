package idk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const keyTTL = 24 * time.Hour

var ErrAlreadyInserted = errors.New("idk already inserted")

type (
	Service interface {
		Insert(ctx context.Context, key uuid.UUID, extraKeys ...string) error
		InsertString(ctx context.Context, key string, extraKeys ...string) error
	}

	Cleaner interface {
		DeleteOutdated(context.Context) error
	}

	Storage interface {
		Insert(ctx context.Context, key uuid.UUID, extraKey string) error
		Delete(ctx context.Context, createdAtBefore time.Time) error
	}

	ServiceImpl struct {
		storage Storage
	}
)

func NewService(storage Storage) ServiceImpl {
	return ServiceImpl{storage: storage}
}

func (s ServiceImpl) Insert(ctx context.Context, key uuid.UUID, extraKeys ...string) error {
	var extraKey string
	if len(extraKeys) > 0 {
		extraKey = strings.Join(extraKeys, "_")
	}

	return s.storage.Insert(ctx, key, extraKey)
}

func (s ServiceImpl) InsertString(ctx context.Context, key string, extraKeys ...string) error { // TODO: optimize using uuid_v5 - idk, key, extra_key - columns
	if len(extraKeys) > 0 {
		key = fmt.Sprintf("%s_%s", key, strings.Join(extraKeys, "_"))
	}

	return s.storage.Insert(ctx, uuid.Nil, key)
}

func (s ServiceImpl) DeleteOutdated(ctx context.Context) error {
	return s.storage.Delete(ctx, time.Now().Add(-keyTTL))
}
