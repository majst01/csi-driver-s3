/*
Copyright 2017 The Kubernetes Authors.

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

package s3

import "testing"

func Test_sanitizeVolumeID(t *testing.T) {
	tests := []struct {
		name     string
		volumeID string
		want     int
	}{
		{
			name:     "smaller",
			volumeID: "abcdef",
			want:     6,
		},
		{
			name:     "eqaul",
			volumeID: "0123456789012345678901234567890123456789012345678901234567890123",
			want:     64,
		},
		{
			name:     "longer",
			volumeID: "0123456789012345678901234567890123456789012345678901234567890123456789",
			want:     64,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := len(sanitizeVolumeID(tt.volumeID)); got != tt.want {
				t.Errorf("sanitizeVolumeID() = %v, want %v", got, tt.want)
			}
		})
	}
}
