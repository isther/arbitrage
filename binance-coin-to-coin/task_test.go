package arbitrage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTask(t *testing.T) {
	assert := assert.New(t)

	task, err := newTask("USDT", "BTC", "BUSD")
	assert.Nil(err)
	assert.Equal(task.StableCoinA, "BUSD")

	task, err = newTask("USDT", "BTC", "TUSD")
	assert.Nil(err)
	assert.Equal(task.StableCoinA, "TUSD")

	task, err = newTask("USDT", "BTC", "USDT")
	assert.NotNil(err)
}

func newTask(
	stableCoinA, tradeCoin, stablecoinB string,
) (*Task, error) {
	return NewArbitrageTask("", "", stableCoinA, tradeCoin, stablecoinB, 0, 0)
}
