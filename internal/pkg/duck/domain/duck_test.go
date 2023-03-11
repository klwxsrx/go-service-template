package domain_test

import (
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDuck_Created_Success(t *testing.T) {
	duckID := uuid.New()
	duck := domain.NewDuck(duckID)

	assert.Equal(t, duckID, duck.ID)
	assert.Len(t, duck.Changes, 1)
	assert.IsType(t, domain.EventDuckCreated{}, duck.Changes[0])
	evt := duck.Changes[0].(domain.EventDuckCreated)
	assert.Equal(t, duckID, evt.DuckID)
}
