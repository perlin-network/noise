package callbacks_test

import (
	"github.com/perlin-network/noise/callbacks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

const (
	numCB = 10
)

func TestReduceCallbacks(t *testing.T) {
	manager := callbacks.NewReduceCallbackManager()

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
	manager := callbacks.NewReduceCallbackManager()

	initial := 3
	expected := initial

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
			if i == numCB/2 {
				return in.(int) + i, callbacks.DeregisterCallback
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
	manager := callbacks.NewReduceCallbackManager()

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
					return in.(int) + i, callbacks.DeregisterCallback
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
	manager := callbacks.NewReduceCallbackManager()

	var actual []int
	var expected []int

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(in interface{}, params ...interface{}) (interface{}, error) {
			actual = append(actual, i)
			return nil, callbacks.DeregisterCallback
		})

		expected = append(expected, i)
	}

	ret, errs := manager.RunCallbacks(nil)

	assert.EqualValues(t, expected, actual, "reduce callbacks failed to execute properly")
	assert.Equal(t, nil, ret, "reduce callbacks for some reason didn't return expected val")
	assert.Empty(t, errs, "reduce callbacks still exist in spite of errors being callbacks.DeregisterCallback")

	ret, errs = manager.RunCallbacks(nil)

	assert.EqualValues(t, expected, actual, "reduce callbacks failed to be de-registered")
	assert.Equal(t, nil, ret, "reduce callbacks for some reason didn't return expected val")
	assert.Empty(t, errs, "reduce callbacks still exist in spite of errors being callbacks.DeregisterCallback")
}

func TestReduceCallbacksOnError(t *testing.T) {
	manager := callbacks.NewReduceCallbackManager()

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

func TestSequentialCallbacks(t *testing.T) {
	manager := callbacks.NewSequentialCallbackManager()

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
	manager := callbacks.NewSequentialCallbackManager()

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
					return callbacks.DeregisterCallback
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
	manager := callbacks.NewSequentialCallbackManager()

	var actual []int
	var expected []int

	for i := 0; i < numCB; i++ {
		i := i

		manager.RegisterCallback(func(params ...interface{}) error {
			actual = append(actual, i)
			return callbacks.DeregisterCallback
		})

		expected = append(expected, i)
	}

	errs := manager.RunCallbacks()

	assert.EqualValues(t, expected, actual, "sequential callbacks failed to execute properly")
	assert.Empty(t, errs, "sequential callbacks still exist in spite of errors being callbacks.DeregisterCallback")

	errs = manager.RunCallbacks()

	assert.EqualValues(t, expected, actual, "sequential callbacks failed to be de-registered")
	assert.Empty(t, errs, "sequential callbacks still exist in spite of errors being callbacks.DeregisterCallback")
}

func TestSequentialCallbacksOnError(t *testing.T) {
	manager := callbacks.NewSequentialCallbackManager()

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

// TODO(kenta): finish tests for opcode callbacks
func TestOpcodeCallback(t *testing.T) {
	{
		// test random order
		var results []int
		ocm := callbacks.NewOpcodeCallbackManager()
		wg := &sync.WaitGroup{}
		for i := 0; i < numCB; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				ocm.RegisterCallback(byte(i), func(params ...interface{}) error {
					results = append(results, i)
					return nil
				})
			}(i)
		}
		wg.Wait()

		// call the callbacks 3 times
		for j := 0; j < 3; j++ {
			results = nil
			for i := 0; i < numCB; i++ {
				errs := ocm.RunCallbacks(byte(i))
				assert.Equal(t, 0, len(errs))
			}
			assert.Equal(t, numCB, len(results))
			//t.Logf("res=%+v", results)
		}
	}
	{
		// test in order
		var results []int
		ocm := callbacks.NewOpcodeCallbackManager()
		for i := 0; i < numCB; i++ {
			j := i
			ocm.RegisterCallback(byte(i), func(params ...interface{}) error {
				results = append(results, j)
				return nil
			})
		}

		// call the callbacks 3 times
		for j := 0; j < 3; j++ {
			results = nil
			for i := 0; i < numCB; i++ {
				errs := ocm.RunCallbacks(byte(i))
				assert.Equal(t, 0, len(errs))
			}
			assert.Equal(t, numCB, len(results))
			for i := 0; i < numCB; i++ {
				assert.Equal(t, i, results[i])
			}
			//t.Logf("res=%+v", results)
		}
	}
	{
		// test errors
		var results []int
		ocm := callbacks.NewOpcodeCallbackManager()
		for i := 0; i < numCB; i++ {
			j := i
			ocm.RegisterCallback(byte(i), func(params ...interface{}) error {
				results = append(results, j)
				//return errors.Errorf("Error-%d", j)
				return callbacks.DeregisterCallback
			})
		}

		// call the callbacks 3 times
		for j := 0; j < 3; j++ {
			results = nil
			for i := 0; i < numCB; i++ {
				errs := ocm.RunCallbacks(byte(i))
				assert.Equal(t, 1, len(errs))
			}
			assert.Equal(t, numCB, len(results))
			for i := 0; i < numCB; i++ {
				assert.Equal(t, i, results[i])
			}
			t.Logf("res=%+v", results)
		}
	}
}
