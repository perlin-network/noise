package callbacks

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestReduceCallbacks(t *testing.T) {
	manager := NewReduceCallbackManager()

	initial := 3
	expected := initial

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
			return in.(int) + i, nil
		})

		expected += i
	}

	actual, errs := manager.RunCallbacks(initial)

	assert.Equal(t, expected, actual, "got invalid reduce result from reduce callbacks")
	assert.Empty(t, errs, "expected no errors from reduce callbacks")
}

func TestReduceCallbacksDeregisterMidway(t *testing.T) {
	manager := NewReduceCallbackManager()

	initial := 3
	expected := initial

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
			if i == numCB/2 {
				return in.(int) + i, DeregisterCallback
			}

			return in.(int) + i, nil
		})

		expected += i
	}

	actual, errs := manager.RunCallbacks(initial)

	assert.Equal(t, expected, actual, "got invalid reduce result from reduce callbacks")
	assert.Empty(t, errs, "expected no errors from reduce callbacks")

	actual, errs = manager.RunCallbacks(initial)

	removedMidwayVal := numCB / 2

	assert.Equal(t, expected-removedMidwayVal, actual, "got invalid reduce result from reduce callbacks after deregistering callback")
	assert.Empty(t, errs, "expected no errors from reduce callbacks")
}

func TestReduceCallbacksConcurrent(t *testing.T) {
	manager := NewReduceCallbackManager()

	initial := 3
	expected := initial

	var wg sync.WaitGroup

	for i := 0; i < numCB; i++ {
		wg.Add(1)

		i := i

		go func() {
			defer wg.Done()

			manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
				if i == numCB/2 {
					return in.(int) + i, DeregisterCallback
				}

				return in.(int) + i, nil
			})
		}()

		expected += i
	}

	wg.Wait()

	actual, errs := manager.RunCallbacks(initial)

	assert.Equal(t, expected, actual, "got invalid reduce result from reduce callbacks")
	assert.Empty(t, errs, "expected no errors from reduce callbacks")

	actual, errs = manager.RunCallbacks(initial)

	removedMidwayVal := numCB / 2

	assert.Equal(t, expected-removedMidwayVal, actual, "got invalid reduce result from reduce callbacks after deregistering callback")
	assert.Empty(t, errs, "expected no errors from reduce callbacks")
}

func TestReduceCallbacksDeregistered(t *testing.T) {
	manager := NewReduceCallbackManager()

	var actual []int
	var expected []int

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
			actual = append(actual, i)
			return nil, DeregisterCallback
		})

		expected = append(expected, i)
	}

	ret, errs := manager.RunCallbacks(nil)

	assert.EqualValues(t, expected, actual, "reduce callbacks failed to execute properly")
	assert.Equal(t, nil, ret, "reduce callbacks for some reason didn't return expected val")
	assert.Empty(t, errs, "reduce callbacks still exist in spite of errors being DeregisterCallback")

	ret, errs = manager.RunCallbacks(nil)

	assert.EqualValues(t, expected, actual, "reduce callbacks failed to be de-registered")
	assert.Equal(t, nil, ret, "reduce callbacks for some reason didn't return expected val")
	assert.Empty(t, errs, "reduce callbacks still exist in spite of errors being DeregisterCallback")
}

/*
func TestReduceCallbacksOnError(t *testing.T) {
	manager := NewReduceCallbackManager()

	var expected []error

	for i := 0; i < numCB; i++ {
		err := errors.Errorf("%d", i)

		manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
			return nil, err
		})

		expected = append(expected, err)
	}

	ret, actual := manager.RunCallbacks(nil)

	assert.Equal(t, nil, ret, "reduce callbacks for some reason didn't return expected val")
	assert.EqualValues(t, expected, actual, "reduce callbacks failed to return errors properly")

	ret, actual = manager.RunCallbacks(nil)

	assert.Equal(t, nil, ret, "reduce callbacks for some reason didn't return expected val")
	assert.Empty(t, actual, "reduce callbacks still exist after errors were returned")
}
*/
