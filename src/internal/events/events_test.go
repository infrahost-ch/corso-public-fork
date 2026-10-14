package events_test

import (
	"testing"

	"github.com/alcionai/clues"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/alcionai/corso/src/internal/events"
	"github.com/alcionai/corso/src/internal/tester"
)

type EventsUnitSuite struct {
	tester.Suite
}

func TestEventsUnitSuite(t *testing.T) {
	suite.Run(t, &EventsUnitSuite{Suite: tester.NewUnitSuite(t)})
}

func (suite *EventsUnitSuite) TestNewBus() {
	t := suite.T()

	ctx, flush := tester.NewContext(t)
	defer flush()

	b := events.NewBus()

	// The bus drops events; this only guards against panics.
	b.Event(ctx, events.BackupEnd, map[string]any{events.BackupID: "id"})

	err := b.Close()
	require.NoError(t, err, clues.ToCore(err))
}
