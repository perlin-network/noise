package callbacks

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync"
	"testing"
)

func TestSequentialCallbacks(t *testing.T) {
	manager := NewSequentialCallbackManager()

	initial := uint32(3)
	actual, expected := initial, initial

	for i := 0; i < numCB; i++ {
		i := uint32(i)

		manager.RegisterCallback(func(params ...interface{}) error {
			actual += i
			return nil
		})

		expected += i
	}

	errs := manager.RunCallbacks(initial)

	assert.Equal(t, expected, actual, "got invalid result from sequential callbacks")
	assert.Empty(t, errs, "expected no errors from sequential callbacks")
}

func TestSequentialCallbacksConcurrent(t *testing.T) {
	manager := NewSequentialCallbackManager()

	initial := uint32(3)
	actual, expected := initial, initial

	var wg sync.WaitGroup

	for i := 0; i < numCB; i++ {
		wg.Add(1)

		i := uint32(i)

		go func() {
			defer wg.Done()

			manager.RegisterCallback(func(params ...interface{}) error {
				actual += i

				if i == numCB/2 {
					return DeregisterCallback
				}

				return nil
			})
		}()

		expected += i
	}

	wg.Wait()

	errs := manager.RunCallbacks()

	assert.Equal(t, expected, actual, "got invalid result from sequential callbacks")
	assert.Empty(t, errs, "expected no errors from sequential callbacks")

	errs = manager.RunCallbacks()

	removedMidwayVal := uint32(numCB / 2)

	assert.Equal(t, expected*2-removedMidwayVal-initial, actual, "got invalid result from sequential callbacks after deregistering callback")
	assert.Empty(t, errs, "expected no errors from sequential callbacks")
}

func TestSequentialCallbackDeregistered(t *testing.T) {
	manager := NewSequentialCallbackManager()

	var actual []int
	var expected []int

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(params ...interface{}) error {
			actual = append(actual, i)
			return DeregisterCallback
		})

		expected = append(expected, i)
	}

	errs := manager.RunCallbacks()

	assert.EqualValues(t, expected, actual, "sequential callbacks failed to execute properly")
	assert.Empty(t, errs, "sequential callbacks still exist in spite of errors being DeregisterCallback")

	errs = manager.RunCallbacks()

	assert.EqualValues(t, expected, actual, "sequential callbacks failed to be de-registered")
	assert.Empty(t, errs, "sequential callbacks still exist in spite of errors being DeregisterCallback")
}

func TestSequentialCallbacksOnError(t *testing.T) {
	manager := NewSequentialCallbackManager()

	var expected []error

	for i := 0; i < numCB; i++ {
		err := errors.Errorf("%d", i)

		manager.RegisterCallback(func(params ...interface{}) error {
			return err
		})

		expected = append(expected, err)
	}

	actual := manager.RunCallbacks()

	assert.EqualValues(t, expected, actual, "sequential callbacks failed to return errors properly")

	actual = manager.RunCallbacks()

	assert.Empty(t, actual, "sequential callbacks still exist after errors were returned")
}

func TestSequentialCallbackIntegration(t *testing.T) {
	var funcs []callback

	removed := make(map[int]struct{})
	var indices []int

	// Create callbacks, and randomly choose ones to deregister.
	for i := 0; i < numCB; i++ {
		i := i

		remove := false

		// 1/3rd chance to remove callback.
		if rand.Intn(3) == 1 {
			remove = true
			removed[i] = struct{}{}
		}

		funcs = append(funcs, func(params ...interface{}) error {
			if remove {
				return DeregisterCallback
			}

			return nil
		})

		indices = append(indices, i)
	}

	manager := NewSequentialCallbackManager()
	for i := 0; i < numCB; i++ {
		manager.RegisterCallback(funcs[i])
	}

	var expected []*wrappedCallback

	for i := 0; i < numCB; i++ {
		if _, deregistered := removed[i]; !deregistered {
			expected = append(expected, manager.callbacks[i])
		}
	}

	// Run once and check everything's de-registered properly.
	errs := manager.RunCallbacks()
	assert.Empty(t, errs, "callbacks unexpectedly returned errs")

	assert.EqualValues(t, expected, manager.callbacks, "callback sequence is unexpected after deregistering")

	// Run twice and check nothing changes.
	errs = manager.RunCallbacks()
	assert.Empty(t, errs, "callbacks unexpectedly returned errs")

	assert.EqualValues(t, expected, manager.callbacks, "callback sequence is unexpected after running after deregistering")
}
