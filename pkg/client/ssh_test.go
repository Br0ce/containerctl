package client

import "testing"

func Test_resolveHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		wantUser string
		wantHost string
		wantErr  bool
	}{
		{"no port", "example.com", "", "example.com:22", false},
		{"with port", "example.com:11", "", "example.com:11", false},
		{"with username", "user@example.com", "user", "example.com:22", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotUser, gotHost, gotErr := resolveHost(test.host)
			if gotErr != nil {
				if !test.wantErr {
					t.Errorf("getAdd() failed: %v", gotErr)
				}
				return
			}
			if test.wantErr {
				t.Fatal("getAdd() succeeded unexpectedly")
			}

			if gotUser != test.wantUser {
				t.Errorf("getAdd() User: got = %v, want %v", gotUser, test.wantUser)
			}

			if gotHost != test.wantHost {
				t.Errorf("getAdd() Host: got = %v, want %v", gotHost, test.wantHost)
			}
		})
	}
}
