// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package gcp

import (
	"context"
	"sync"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/deploymentmanager/v2"
)

// Ensure, that ClientMock does implement Client.
// If this is not the case, regenerate this file with moq.
var _ Client = &ClientMock{}

// ClientMock is a mock implementation of Client.
//
//	func TestSomethingThatUsesClient(t *testing.T) {
//
//		// make and configure a mocked Client
//		mockedClient := &ClientMock{
//			DeleteFunc: func(ctx context.Context, coreInstanceName string) error {
//				panic("mock out the Delete method")
//			},
//			DeployFunc: func(contextMoqParam context.Context) error {
//				panic("mock out the Deploy method")
//			},
//			FollowOperationsFunc: func(contextMoqParam context.Context) (*deploymentmanager.Operation, error) {
//				panic("mock out the FollowOperations method")
//			},
//			GetInstanceFunc: func(ctx context.Context, zone string, instance string) (*compute.Instance, error) {
//				panic("mock out the GetInstance method")
//			},
//			RollbackFunc: func(contextMoqParam context.Context) error {
//				panic("mock out the Rollback method")
//			},
//			SetConfigFunc: func(newConfig Config)  {
//				panic("mock out the SetConfig method")
//			},
//		}
//
//		// use mockedClient in code that requires Client
//		// and then make assertions.
//
//	}
type ClientMock struct {
	// DeleteFunc mocks the Delete method.
	DeleteFunc func(ctx context.Context, coreInstanceName string) error

	// DeployFunc mocks the Deploy method.
	DeployFunc func(contextMoqParam context.Context) error

	// FollowOperationsFunc mocks the FollowOperations method.
	FollowOperationsFunc func(contextMoqParam context.Context) (*deploymentmanager.Operation, error)

	// GetInstanceFunc mocks the GetInstance method.
	GetInstanceFunc func(ctx context.Context, zone string, instance string) (*compute.Instance, error)

	// RollbackFunc mocks the Rollback method.
	RollbackFunc func(contextMoqParam context.Context) error

	// SetConfigFunc mocks the SetConfig method.
	SetConfigFunc func(newConfig Config)

	// calls tracks calls to the methods.
	calls struct {
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// CoreInstanceName is the coreInstanceName argument value.
			CoreInstanceName string
		}
		// Deploy holds details about calls to the Deploy method.
		Deploy []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
		// FollowOperations holds details about calls to the FollowOperations method.
		FollowOperations []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
		// GetInstance holds details about calls to the GetInstance method.
		GetInstance []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Zone is the zone argument value.
			Zone string
			// Instance is the instance argument value.
			Instance string
		}
		// Rollback holds details about calls to the Rollback method.
		Rollback []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
		// SetConfig holds details about calls to the SetConfig method.
		SetConfig []struct {
			// NewConfig is the newConfig argument value.
			NewConfig Config
		}
	}
	lockDelete           sync.RWMutex
	lockDeploy           sync.RWMutex
	lockFollowOperations sync.RWMutex
	lockGetInstance      sync.RWMutex
	lockRollback         sync.RWMutex
	lockSetConfig        sync.RWMutex
}

// Delete calls DeleteFunc.
func (mock *ClientMock) Delete(ctx context.Context, coreInstanceName string) error {
	if mock.DeleteFunc == nil {
		panic("ClientMock.DeleteFunc: method is nil but Client.Delete was just called")
	}
	callInfo := struct {
		Ctx              context.Context
		CoreInstanceName string
	}{
		Ctx:              ctx,
		CoreInstanceName: coreInstanceName,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(ctx, coreInstanceName)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//
//	len(mockedClient.DeleteCalls())
func (mock *ClientMock) DeleteCalls() []struct {
	Ctx              context.Context
	CoreInstanceName string
} {
	var calls []struct {
		Ctx              context.Context
		CoreInstanceName string
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// Deploy calls DeployFunc.
func (mock *ClientMock) Deploy(contextMoqParam context.Context) error {
	if mock.DeployFunc == nil {
		panic("ClientMock.DeployFunc: method is nil but Client.Deploy was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
	}{
		ContextMoqParam: contextMoqParam,
	}
	mock.lockDeploy.Lock()
	mock.calls.Deploy = append(mock.calls.Deploy, callInfo)
	mock.lockDeploy.Unlock()
	return mock.DeployFunc(contextMoqParam)
}

// DeployCalls gets all the calls that were made to Deploy.
// Check the length with:
//
//	len(mockedClient.DeployCalls())
func (mock *ClientMock) DeployCalls() []struct {
	ContextMoqParam context.Context
} {
	var calls []struct {
		ContextMoqParam context.Context
	}
	mock.lockDeploy.RLock()
	calls = mock.calls.Deploy
	mock.lockDeploy.RUnlock()
	return calls
}

// FollowOperations calls FollowOperationsFunc.
func (mock *ClientMock) FollowOperations(contextMoqParam context.Context) (*deploymentmanager.Operation, error) {
	if mock.FollowOperationsFunc == nil {
		panic("ClientMock.FollowOperationsFunc: method is nil but Client.FollowOperations was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
	}{
		ContextMoqParam: contextMoqParam,
	}
	mock.lockFollowOperations.Lock()
	mock.calls.FollowOperations = append(mock.calls.FollowOperations, callInfo)
	mock.lockFollowOperations.Unlock()
	return mock.FollowOperationsFunc(contextMoqParam)
}

// FollowOperationsCalls gets all the calls that were made to FollowOperations.
// Check the length with:
//
//	len(mockedClient.FollowOperationsCalls())
func (mock *ClientMock) FollowOperationsCalls() []struct {
	ContextMoqParam context.Context
} {
	var calls []struct {
		ContextMoqParam context.Context
	}
	mock.lockFollowOperations.RLock()
	calls = mock.calls.FollowOperations
	mock.lockFollowOperations.RUnlock()
	return calls
}

// GetInstance calls GetInstanceFunc.
func (mock *ClientMock) GetInstance(ctx context.Context, zone string, instance string) (*compute.Instance, error) {
	if mock.GetInstanceFunc == nil {
		panic("ClientMock.GetInstanceFunc: method is nil but Client.GetInstance was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		Zone     string
		Instance string
	}{
		Ctx:      ctx,
		Zone:     zone,
		Instance: instance,
	}
	mock.lockGetInstance.Lock()
	mock.calls.GetInstance = append(mock.calls.GetInstance, callInfo)
	mock.lockGetInstance.Unlock()
	return mock.GetInstanceFunc(ctx, zone, instance)
}

// GetInstanceCalls gets all the calls that were made to GetInstance.
// Check the length with:
//
//	len(mockedClient.GetInstanceCalls())
func (mock *ClientMock) GetInstanceCalls() []struct {
	Ctx      context.Context
	Zone     string
	Instance string
} {
	var calls []struct {
		Ctx      context.Context
		Zone     string
		Instance string
	}
	mock.lockGetInstance.RLock()
	calls = mock.calls.GetInstance
	mock.lockGetInstance.RUnlock()
	return calls
}

// Rollback calls RollbackFunc.
func (mock *ClientMock) Rollback(contextMoqParam context.Context) error {
	if mock.RollbackFunc == nil {
		panic("ClientMock.RollbackFunc: method is nil but Client.Rollback was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
	}{
		ContextMoqParam: contextMoqParam,
	}
	mock.lockRollback.Lock()
	mock.calls.Rollback = append(mock.calls.Rollback, callInfo)
	mock.lockRollback.Unlock()
	return mock.RollbackFunc(contextMoqParam)
}

// RollbackCalls gets all the calls that were made to Rollback.
// Check the length with:
//
//	len(mockedClient.RollbackCalls())
func (mock *ClientMock) RollbackCalls() []struct {
	ContextMoqParam context.Context
} {
	var calls []struct {
		ContextMoqParam context.Context
	}
	mock.lockRollback.RLock()
	calls = mock.calls.Rollback
	mock.lockRollback.RUnlock()
	return calls
}

// SetConfig calls SetConfigFunc.
func (mock *ClientMock) SetConfig(newConfig Config) {
	if mock.SetConfigFunc == nil {
		panic("ClientMock.SetConfigFunc: method is nil but Client.SetConfig was just called")
	}
	callInfo := struct {
		NewConfig Config
	}{
		NewConfig: newConfig,
	}
	mock.lockSetConfig.Lock()
	mock.calls.SetConfig = append(mock.calls.SetConfig, callInfo)
	mock.lockSetConfig.Unlock()
	mock.SetConfigFunc(newConfig)
}

// SetConfigCalls gets all the calls that were made to SetConfig.
// Check the length with:
//
//	len(mockedClient.SetConfigCalls())
func (mock *ClientMock) SetConfigCalls() []struct {
	NewConfig Config
} {
	var calls []struct {
		NewConfig Config
	}
	mock.lockSetConfig.RLock()
	calls = mock.calls.SetConfig
	mock.lockSetConfig.RUnlock()
	return calls
}
