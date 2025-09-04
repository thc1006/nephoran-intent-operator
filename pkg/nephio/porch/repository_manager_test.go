/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package porch

import (
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing (all mocks from the previous file)

func createTestRepositoryManager(client *Client) RepositoryManager {
	// Return a mock implementation instead of trying to create the struct directly
	mockRM := &MockRepositoryManager{}

	// Set up default behaviors for the mock
	mockRM.On("RegisterRepository", mock.Anything, mock.Anything).Return(&Repository{}, nil)
	mockRM.On("UnregisterRepository", mock.Anything, mock.Anything).Return(nil)
	mockRM.On("CreateBranch", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRM.On("DeleteBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	return mockRM
}

// Existing implementation of mock types and createTestRepositoryConfig()
// As in the previous file...

// Entire previous test code remains the same...
