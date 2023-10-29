package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/klwxsrx/go-service-template/internal/duck/domain"
)

func TestNewDuck_Created_Success(t *testing.T) {
	duckID := uuid.New()
	name := "SomeDuckName"

	duck := domain.NewDuck(duckID, name)

	assert.Equal(t, duckID, duck.ID)
	assert.Len(t, duck.Changes, 1)
	assert.IsType(t, domain.EventDuckCreated{}, duck.Changes[0])
	evt := duck.Changes[0].(domain.EventDuckCreated)
	assert.Equal(t, duckID, evt.DuckID)
}
