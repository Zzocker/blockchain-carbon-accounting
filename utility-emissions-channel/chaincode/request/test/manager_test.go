package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagerInit(t *testing.T) {
	managerCC := prepareManager()
	is := assert.New(t)

	channelName := "TestChannel"
	managerCC.ChannelID = channelName
	managerCC.MockInit("init", nil)

	is.Equal(1, len(managerCC.State))

	for _, v := range managerCC.State {
		is.Equal(channelName, string(v))
	}
}
