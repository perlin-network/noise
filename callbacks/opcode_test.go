package callbacks

import (
	"testing"
)

// TODO(kenta): finish tests for opcode callbacks
func TestOpcodeCallback(t *testing.T) {
	//{
	//	// test random order
	//	var results []int
	//	ocm := NewOpcodeCallbackManager()
	//	wg := &sync.WaitGroup{}
	//	for i := 0; i < numCB; i++ {
	//		wg.Add(1)
	//		go func(i int) {
	//			defer wg.Done()
	//			ocm.RegisterCallback(byte(i), func(params ...interface{}) error {
	//				results = append(results, i)
	//				return nil
	//			})
	//		}(i)
	//	}
	//	wg.Wait()
	//
	//	// call the callbacks 3 times
	//	for j := 0; j < 3; j++ {
	//		results = nil
	//		for i := 0; i < numCB; i++ {
	//			errs := ocm.RunCallbacks(byte(i))
	//			assert.Equal(t, 0, len(errs))
	//		}
	//		assert.Equal(t, numCB, len(results))
	//		//t.Logf("res=%+v", results)
	//	}
	//}
	//{
	//	// test in order
	//	var results []int
	//	ocm := NewOpcodeCallbackManager()
	//	for i := 0; i < numCB; i++ {
	//		j := i
	//		ocm.RegisterCallback(byte(i), func(params ...interface{}) error {
	//			results = append(results, j)
	//			return nil
	//		})
	//	}
	//
	//	// call the callbacks 3 times
	//	for j := 0; j < 3; j++ {
	//		results = nil
	//		for i := 0; i < numCB; i++ {
	//			errs := ocm.RunCallbacks(byte(i))
	//			assert.Equal(t, 0, len(errs))
	//		}
	//		assert.Equal(t, numCB, len(results))
	//		for i := 0; i < numCB; i++ {
	//			assert.Equal(t, i, results[i])
	//		}
	//		//t.Logf("res=%+v", results)
	//	}
	//}
	//{
	//	// test errors
	//	var results []int
	//	ocm := NewOpcodeCallbackManager()
	//	for i := 0; i < numCB; i++ {
	//		j := i
	//		ocm.RegisterCallback(byte(i), func(params ...interface{}) error {
	//			results = append(results, j)
	//			//return errors.Errorf("Error-%d", j)
	//			return DeregisterCallback
	//		})
	//	}
	//
	//	// call the callbacks 3 times
	//	for j := 0; j < 3; j++ {
	//		results = nil
	//		for i := 0; i < numCB; i++ {
	//			errs := ocm.RunCallbacks(byte(i))
	//			assert.Equal(t, 0, len(errs))
	//		}
	//		assert.Equal(t, numCB, len(results))
	//		for i := 0; i < numCB; i++ {
	//			assert.Equal(t, i, results[i])
	//		}
	//		t.Logf("res=%+v", results)
	//	}
	//}
}
